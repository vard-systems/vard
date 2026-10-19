package usersessions

import (
	"github.com/google/uuid"

	"loginer/internal/store"
)

func (r listRequest) query(limit int) store.SessionQuery {
	q := store.SessionQuery{Search: r.Search, Limit: limit}

	if r.User != "" {
		q.UserID = uuid.MustParse(r.User)
	}
	if r.After != "" {
		q.After = uuid.MustParse(r.After)
	}

	return q
}
