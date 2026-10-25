package model

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Every row gets its id from the BeforeCreate hook, not from the database.
func TestBeforeCreateSetsID(t *testing.T) {
	admin := &Admin{Email: "a@b.test"}

	if err := admin.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}

	if admin.ID == uuid.Nil {
		t.Fatal("ID is still nil after BeforeCreate")
	}
	if got := admin.ID.Version(); got != 7 {
		t.Errorf("uuid version = %d, want 7 (v7 ids sort by creation time)", got)
	}
}

// An id set by the caller must survive, so a row can be created with a known
// id.
func TestBeforeCreateKeepsExistingID(t *testing.T) {
	want := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	admin := &Admin{ID: want}

	if err := admin.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}

	if admin.ID != want {
		t.Errorf("ID = %s, want the id it was given (%s)", admin.ID, want)
	}
}

// Version 7 ids carry a timestamp in their leading bits, so ids made later
// sort after ids made earlier.
func TestIDsSortByCreationOrder(t *testing.T) {
	var previous string

	for i := range 50 {
		row := &Admin{}
		if err := row.BeforeCreate(nil); err != nil {
			t.Fatal(err)
		}

		if id := row.ID.String(); id <= previous && i > 0 {
			t.Fatalf("id %s does not sort after %s", id, previous)
		} else {
			previous = id
		}
	}
}

// AuditLog does not embed Base, so it carries its own hook.
func TestAuditLogBeforeCreateSetsID(t *testing.T) {
	entry := &AuditLog{Action: "admin_users.create"}

	if err := entry.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}

	if entry.ID == uuid.Nil {
		t.Fatal("ID is still nil after BeforeCreate")
	}
}

// All feeds both the migration generator and AutoMigrate; a model missing
// from it silently never gets a table.
func TestAllListsEveryModel(t *testing.T) {
	if got, want := len(All()), 47; got != want {
		t.Errorf("All() has %d models, want %d — was a new model added without listing it here?", got, want)
	}
}

// A bool column must not default to true. GORM leaves a zero value out of an
// INSERT when the column has a default, so false is silently stored as true:
// an application created disabled comes out enabled, a user created inactive
// comes out active. Callers set these explicitly instead.
func TestNoBoolDefaultsToTrue(t *testing.T) {
	for _, m := range All() {
		typ := reflect.TypeOf(m).Elem()

		for i := range typ.NumField() {
			field := typ.Field(i)
			if field.Type.Kind() != reflect.Bool {
				continue
			}

			if strings.Contains(field.Tag.Get("gorm"), "default:true") {
				t.Errorf("%s.%s has default:true, so false cannot be stored", typ.Name(), field.Name)
			}
		}
	}
}
