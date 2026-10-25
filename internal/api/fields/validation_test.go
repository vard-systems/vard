package fields

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"vard/internal/api/kit/respond"
	"vard/internal/model"
)

// status is the status a fault carries, or 0 when the error is not one.
func status(err error) int {
	if fault, ok := errors.AsType[respond.Fault](err); ok {
		return fault.Status
	}

	return 0
}

// TestNewField covers what a create request has to say before a field is made
// of it: the name is a column name, and the type is one we know.
func TestNewField(t *testing.T) {
	tests := []struct {
		name       string
		request    fieldRequest
		wantStatus int
	}{
		{
			name:    "a plain text field",
			request: fieldRequest{Name: "nickname", Type: "text", rulesRequest: rulesRequest{Label: "Nickname"}},
		},
		{
			name:    "the name is trimmed and lowered",
			request: fieldRequest{Name: "  Nickname  ", Type: "text"},
		},
		{
			name:       "a name that starts with a digit",
			request:    fieldRequest{Name: "1st_name", Type: "text"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "a name with a dash",
			request:    fieldRequest{Name: "nick-name", Type: "text"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no name at all",
			request:    fieldRequest{Type: "text"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "a type nobody has heard of",
			request:    fieldRequest{Name: "nickname", Type: "blob"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "a rule the type cannot keep",
			request:    fieldRequest{Name: "active", Type: "bool", rulesRequest: rulesRequest{StartsWith: "+"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "a label nobody could read",
			request:    fieldRequest{Name: "nickname", Type: "text", rulesRequest: rulesRequest{Label: strings.Repeat("x", 101)}},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			field, err := request.newField()

			if got := status(err); got != tt.wantStatus {
				t.Fatalf("newField() = status %d (%v), want %d", got, err, tt.wantStatus)
			}

			if tt.wantStatus != 0 {
				return
			}

			if field.Name != "nickname" && field.Name != "active" {
				t.Errorf("name = %q, want it trimmed and lowered", field.Name)
			}
		})
	}
}

// A field with no label of its own is called by its name, rather than being
// left blank in the panel.
func TestApplyToFallsBackToTheName(t *testing.T) {
	field := &model.UserField{Name: "phone", Type: model.FieldText}

	request := rulesRequest{Label: "   "}
	if err := request.applyTo(field); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	if field.Label != "phone" {
		t.Errorf("label = %q, want %q", field.Label, "phone")
	}
}

// An update carries the rules and nothing else: a field's name and type
// belong to the records already stored under them, so they are not on the
// request at all and cannot be moved by one.
func TestApplyToLeavesNameAndType(t *testing.T) {
	field := &model.UserField{Name: "phone", Type: model.FieldText, Label: "Phone"}

	request := rulesRequest{Label: "Phone number"}
	if err := request.applyTo(field); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	if field.Name != "phone" || field.Type != model.FieldText {
		t.Errorf("field = %s/%s, want phone/text", field.Name, field.Type)
	}
	if field.Label != "Phone number" {
		t.Errorf("label = %q, want it updated", field.Label)
	}
}

// Every type the model knows is a type the create endpoint accepts: the rule
// is registered from model.FieldType, so the two cannot drift apart.
func TestNewFieldAcceptsEveryKnownType(t *testing.T) {
	for _, known := range model.FieldTypes {
		t.Run(string(known), func(t *testing.T) {
			request := fieldRequest{Name: "probe", Type: string(known)}

			if _, err := request.newField(); err != nil {
				t.Errorf("newField() = %v, want it accepted", err)
			}
		})
	}
}
