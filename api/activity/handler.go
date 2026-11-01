// Package activity serves the activity log to the panel: the overview, the
// filtered log and its CSV export.
package activity

import (
	"encoding/csv"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
	"vard/internal/interfaces/api/kit/query"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/session"
)

const (
	dashboardEntries = 12
	topActors        = 5
	defaultDays      = 14

	defaultLimit = 50
	maxLimit     = 200

	exportLimit = 10000
	exportPage  = 500
)

var ranges = map[int]bool{7: true, 14: true, 30: true, 90: true}

type Handler struct {
	store *store.Store
	log   *slog.Logger
}

func New(st *store.Store, log *slog.Logger) *Handler {
	return &Handler{store: st, log: log}
}

// Overview is the panel's front page for a range of days: totals, sign-in
// outcomes, daily activity, the busiest people and the latest entries. Its five
// reads run concurrently.
func (h *Handler) Overview(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()

	days := query.Int(c, "days", defaultDays, 90)
	if !ranges[days] {
		days = defaultDays
	}
	since := now.AddDate(0, 0, -days)

	var (
		counts  store.Counts
		signIns store.SignIns
		daily   []store.DayCount
		actors  []store.ActorCount
		events  []model.AuditLog
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() (err error) { counts, err = h.store.Counts(groupCtx, now, since); return })
	group.Go(func() (err error) { signIns, err = h.store.SignInsSince(groupCtx, since); return })
	group.Go(func() (err error) { daily, err = h.store.DailyActivity(groupCtx, now, days); return })
	group.Go(func() (err error) { actors, err = h.store.TopActors(groupCtx, since, topActors); return })
	group.Go(func() (err error) { events, err = h.store.AuditLog(groupCtx, dashboardEntries); return })

	if err := group.Wait(); err != nil {
		respond.Failure(c, h.log, err, "reading the overview failed")
		return
	}

	activity, err := h.describe(c, events)
	if err != nil {
		respond.Failure(c, h.log, err, "naming what the activity was about failed")
		return
	}

	c.JSON(http.StatusOK, overviewResponse{
		Days:      days,
		Counts:    counts,
		SignIns:   signIns,
		Daily:     daily,
		TopActors: actors,
		Activity:  activity,
	})
}

// Logs lists one page of the activity log, newest first, narrowed by the
// filters parseFilter reads, with the cursor the next page starts from.
func (h *Handler) Logs(c *gin.Context) {
	filter, err := parseFilter(c, query.Int(c, "limit", defaultLimit, maxLimit))
	if err != nil {
		respond.Failure(c, h.log, err, "reading the log filters failed")
		return
	}

	events, next, err := h.store.AuditQuery(c.Request.Context(), filter)
	if err != nil {
		respond.Failure(c, h.log, err, "listing logs failed")
		return
	}

	described, err := h.describe(c, events)
	if err != nil {
		respond.Failure(c, h.log, err, "naming what the logs were about failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": newLogResponses(events, described),
		"next": formatCursor(next),
	})
}

// Export writes matching entries as CSV, newest first, up to exportLimit,
// hiding exactly what the logs page hides.
//
// The first page is read before anything is sent, so a failure to read it is
// an error answer rather than an empty file. Once the file has begun, a later
// failure can only end it early; that is logged, and so is a client that went
// away, which also stops the reading.
func (h *Handler) Export(c *gin.Context) {
	filter, err := parseFilter(c, exportPage)
	if err != nil {
		respond.Failure(c, h.log, err, "reading the log filters failed")
		return
	}

	events, next, err := h.store.AuditQuery(c.Request.Context(), filter)
	if err != nil {
		respond.Failure(c, h.log, err, "exporting logs failed")
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition",
		`attachment; filename="activity-`+time.Now().Format("2006-01-02")+`.csv"`)

	out := csv.NewWriter(c.Writer)
	_ = out.Write([]string{"time", "actor", "action", "target_type", "target_id", "target", "detail", "ip", "user_agent"})

	for written := 0; ; {
		described, err := h.describe(c, events)
		if err != nil {
			h.log.Error("naming what the export was about failed", "error", err)
			break
		}

		for i, event := range events {
			if written == exportLimit {
				break
			}

			row := described[i]
			targetType, targetID, target := "", "", ""
			if row.Target != nil {
				targetType, targetID, target = row.Target.Type, row.Target.ID, row.Target.Name
			}

			_ = out.Write(csvSafe([]string{
				event.CreatedAt.UTC().Format(time.RFC3339), row.Actor, row.Action,
				targetType, targetID, target, row.Detail, row.IP, event.UserAgent,
			}))
			written++
		}

		// csv.Writer keeps its first error; one here is the connection.
		out.Flush()
		if err := out.Error(); err != nil {
			h.log.Warn("the log export stopped: the client went away", "error", err)
			return
		}

		if next == nil || written == exportLimit {
			break
		}

		filter.Before = next
		if events, next, err = h.store.AuditQuery(c.Request.Context(), filter); err != nil {
			h.log.Error("exporting logs failed part way; the file is cut short", "error", err)
			break
		}
	}

	out.Flush()
}

// csvSafe prefixes cells a spreadsheet would read as a formula with a quote.
// User-Agent and email are attacker-controlled.
func csvSafe(row []string) []string {
	for i, cell := range row {
		if cell != "" && strings.ContainsRune("=+-@\t\r", rune(cell[0])) {
			row[i] = "'" + cell
		}
	}

	return row
}

func (h *Handler) describe(c *gin.Context, events []model.AuditLog) ([]eventResponse, error) {
	ids := map[string][]string{}
	for _, event := range events {
		if event.TargetType != "" && event.TargetID != "" {
			ids[event.TargetType] = append(ids[event.TargetType], event.TargetID)
		}
	}

	names := map[string]map[string]store.TargetName{}
	for targetType, of := range ids {
		found, err := h.store.TargetNames(c.Request.Context(), targetType, of)
		if err != nil {
			return nil, err
		}
		names[targetType] = found
	}

	out := make([]eventResponse, 0, len(events))
	for _, event := range events {
		response := newEventResponse(event)

		if event.TargetType != "" {
			response.Target = &targetResponse{Type: event.TargetType, ID: event.TargetID}

			if name, ok := names[event.TargetType][event.TargetID]; ok && maySeeName(c, event.TargetType, name) {
				response.Target.Name = name.Name
			}
		}

		// What a user did at the sign-in pages is recorded under their own
		// address, which is only for administrators who may read users.
		if event.AdminID == nil && event.TargetType == "user" && !maySeeName(c, "user", store.TargetName{}) {
			response.Actor = "A user"
		}

		response.Detail = detail(c, event)
		out = append(out, response)
	}

	return out, nil
}

// maySeeName reports whether the administrator may see a target's name: that is
// for whoever may read that kind of record.
func maySeeName(c *gin.Context, targetType string, name store.TargetName) bool {
	admin := session.Admin(c)
	if admin == nil {
		return false
	}

	switch targetType {
	case "user", "user_field":
		return admin.HasPermission(model.PermUsersRead)
	case "application":
		return name.ApplicationID != nil && session.Allowed(c, model.PermApplicationsRead, *name.ApplicationID)
	case "user_role":
		return name.ApplicationID == nil || session.Allowed(c, model.PermApplicationsRead, *name.ApplicationID)
	case "api":
		return admin.HasPermission(model.PermAPIsRead)
	case "admin_user", "admin_role":
		return admin.IsSuperAdmin()
	case "organization":
		return admin.HasPermission(model.PermOrganizationRead)
	case "login_flow":
		return admin.HasPermission(model.PermLoginFlowsRead)
	case "language":
		return admin.HasPermission(model.PermLanguagesRead)
	case "sso_connection":
		return admin.HasPermission(model.PermSSORead)
	case "social_provider":
		return admin.HasPermission(model.PermSocialRead)
	default:
		return false
	}
}

func detail(c *gin.Context, event model.AuditLog) string {
	text := func(key string) string {
		value, _ := event.Metadata[key].(string)
		return value
	}

	list := func(key string) string {
		values, _ := event.Metadata[key].([]any)

		parts := make([]string, 0, len(values))
		for _, value := range values {
			if part, ok := value.(string); ok {
				parts = append(parts, part)
			}
		}

		return strings.Join(parts, ", ")
	}

	switch event.Action {
	case "admin.login_failed", "admin.login_blocked", "admin.mfa_failed", "user.login_failed", "user.login_blocked":
		return text("reason")
	case "application.api_authorized", "application.api_revoked":
		if admin := session.Admin(c); admin != nil && admin.HasPermission(model.PermAPIsRead) {
			return text("api")
		}
	case "organization.updated":
		if admin := session.Admin(c); admin != nil && admin.HasPermission(model.PermOrganizationRead) {
			return list("fields")
		}
	case "sso_connection.sign_in_failed":
		if admin := session.Admin(c); admin != nil && admin.HasPermission(model.PermSSORead) {
			return text("step") + ": " + text("reason")
		}
	case "language.translated":
		if admin := session.Admin(c); admin != nil && admin.HasPermission(model.PermLanguagesRead) {
			return text("app")
		}
	}

	return ""
}
