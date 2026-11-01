package activity

import (
	"time"

	"vard/core/model"
	"vard/core/store"
)

// targetResponse is what an entry happened to. Name is left out when the
// record is gone or the administrator may not see what it is called.
type targetResponse struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// eventResponse is one entry of the activity list.
type eventResponse struct {
	ID        string          `json:"id"`
	Action    string          `json:"action"`
	Actor     string          `json:"actor"`
	IP        string          `json:"ip"`
	Target    *targetResponse `json:"target"`
	Detail    string          `json:"detail,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// logResponse is the same entry on the logs page, which has room for more.
type logResponse struct {
	eventResponse
	UserAgent string `json:"user_agent"`
}

type overviewResponse struct {
	// Days is the range everything below was counted over.
	Days      int                `json:"days"`
	Counts    store.Counts       `json:"counts"`
	SignIns   store.SignIns      `json:"sign_ins"`
	Daily     []store.DayCount   `json:"daily"`
	TopActors []store.ActorCount `json:"top_actors"`
	Activity  []eventResponse    `json:"activity"`
}

func newEventResponse(event model.AuditLog) eventResponse {
	return eventResponse{
		ID:        event.ID.String(),
		Action:    event.Action,
		Actor:     event.ActorEmail,
		IP:        event.IP,
		CreatedAt: event.CreatedAt,
	}
}

func newLogResponses(events []model.AuditLog, described []eventResponse) []logResponse {
	out := make([]logResponse, 0, len(events))
	for i, event := range events {
		out = append(out, logResponse{eventResponse: described[i], UserAgent: event.UserAgent})
	}

	return out
}
