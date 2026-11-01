package usersessions

import (
	"net/http"

	"vard/internal/interfaces/api/kit/respond"
)

// targetType is what the activity log records these actions against: the
// user whose sessions they were.
const targetType = "user"

// sessionAbsent is a session id that names no session.
var sessionAbsent = respond.Define(http.StatusNotFound, "session_not_found", respond.Admin)

// pageSize is how many sessions one page lists, and pageMax the most a
// request may ask for.
const (
	pageSize = 50
	pageMax  = 200
)

// listRequest is the Sessions page's query string: an address to search by
// its start, one user's sessions, and the session to continue after.
type listRequest struct {
	Search string `form:"search" validate:"max=255"`
	User   string `form:"user" validate:"omitempty,uuid"`
	After  string `form:"after" validate:"omitempty,uuid"`
}
