package activity

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/api/kit/respond"
	"vard/api/kit/session"
	"vard/core/model"
	"vard/core/store"
)

var filterInvalid = respond.Define(http.StatusBadRequest, "logs_filter_invalid", respond.Admin)

// actionPattern is "<resource>.<action>"; names are checked and lists capped
// before reaching a query.
var actionPattern = regexp.MustCompile(`^[a-z_]{1,64}\.[a-z_]{1,64}$`)

const maxActions = 100

const dayLayout = "2006-01-02"

func parseFilter(c *gin.Context, limit int) (store.AuditFilter, error) {
	filter := store.AuditFilter{
		Search: strings.TrimSpace(c.Query("q")),
		Actor:  strings.TrimSpace(c.Query("actor")),
		Limit:  limit,
	}

	if admin := session.Admin(c); admin == nil || !admin.HasPermission(model.PermUsersRead) {
		filter.HideUserActors = true
	}

	actions := c.QueryArray("action")
	if len(actions) > maxActions {
		return filter, filterInvalid.With()
	}
	for _, action := range actions {
		if !actionPattern.MatchString(action) {
			return filter, filterInvalid.With()
		}
	}
	filter.Actions = actions

	var err error
	if filter.From, err = moment(c.Query("from"), false); err != nil {
		return filter, filterInvalid.With()
	}
	if filter.To, err = moment(c.Query("to"), true); err != nil {
		return filter, filterInvalid.With()
	}

	if raw := c.Query("before"); raw != "" {
		cursor, err := parseCursor(raw)
		if err != nil {
			return filter, filterInvalid.With()
		}
		filter.Before = &cursor
	}

	return filter, nil
}

func moment(raw string, end bool) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}

	if day, err := time.ParseInLocation(dayLayout, raw, time.Local); err == nil {
		if end {
			return day.AddDate(0, 0, 1), nil
		}
		return day, nil
	}

	return time.Parse(time.RFC3339, raw)
}

func formatCursor(cursor *store.AuditCursor) string {
	if cursor == nil {
		return ""
	}

	return strconv.FormatInt(cursor.At.UnixNano(), 10) + "_" + cursor.ID.String()
}

func parseCursor(raw string) (store.AuditCursor, error) {
	at, id, ok := strings.Cut(raw, "_")
	if !ok {
		return store.AuditCursor{}, strconv.ErrSyntax
	}

	nanos, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		return store.AuditCursor{}, err
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return store.AuditCursor{}, err
	}

	return store.AuditCursor{At: time.Unix(0, nanos), ID: parsed}, nil
}
