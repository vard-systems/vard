package store

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vard/core/database/databasetest"
	"vard/core/model"
)

// A write that spans tables happens whole or not at all: a user whose role
// cannot be given is not left behind without it, and likewise a role or an
// API whose scopes cannot be stored.
func TestLiveAWriteAcrossTablesIsWholeOrNothing(t *testing.T) {
	db, _ := databasetest.Open(t)
	st := New(db)
	ctx := context.Background()
	nowhere := model.UserRole{Base: model.Base{ID: uuid.New()}, Name: "ghost"}
	tooLong := model.APIScope{Name: strings.Repeat("s", 300)}

	tests := []struct {
		name  string
		write func() error
		table string
	}{
		{"a user given a role that does not exist", func() error {
			return st.CreateUser(ctx, &model.User{Email: "ada@example.com", IsActive: true, Roles: []model.UserRole{nowhere}})
		}, "users"},
		{"a role including one that does not exist", func() error {
			return st.CreateUserRole(ctx, &model.UserRole{Name: "editor", Inherits: []model.UserRole{nowhere}})
		}, "user_roles"},
		{"an API with a scope that cannot be stored", func() error {
			return st.CreateAPI(ctx, &model.API{
				Name: "Orders", Identifier: "https://orders.example", SigningAlgorithm: model.AlgRS256,
				Scopes: []model.APIScope{tooLong},
			})
		}, "apis"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := count(t, db, tt.table)
			if err := tt.write(); err == nil {
				t.Fatal("the write succeeded")
			}
			if after := count(t, db, tt.table); after != before {
				t.Errorf("%s went from %d rows to %d: the half that worked was kept", tt.table, before, after)
			}
		})
	}
}

func count(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
