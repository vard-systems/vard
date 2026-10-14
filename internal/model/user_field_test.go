package model

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// field builds a field of the given type for a test to use.
func field(t FieldType, required bool) UserField {
	return UserField{Name: "probe", Label: "Probe", Type: t, IsRequired: required}
}

// TestNormalize covers what a submitted value becomes, which is the one place
// a field's type actually means something.
func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		field UserField
		input any
		want  any
	}{
		{name: "text is trimmed", field: field(FieldText, false), input: "  Mira  ", want: "Mira"},
		{name: "number from a string", field: field(FieldNumber, false), input: "1250", want: 1250.0},
		{name: "number from a number", field: field(FieldNumber, false), input: 42.5, want: 42.5},
		{name: "bool from a bool", field: field(FieldBool, false), input: true, want: true},
		{name: "bool from a string", field: field(FieldBool, false), input: "true", want: true},
		{name: "email is trimmed", field: field(FieldEmail, false), input: " a@b.com ", want: "a@b.com"},
		{name: "a day becomes a timestamp", field: field(FieldDate, false), input: "2026-09-13", want: "2026-09-13T00:00:00Z"},

		// An optional field with nothing in it is stored as nothing, rather
		// than as an empty string that would have to be handled everywhere.
		{name: "empty optional text", field: field(FieldText, false), input: "", want: nil},
		{name: "missing optional text", field: field(FieldText, false), input: nil, want: nil},
		{name: "empty optional number", field: field(FieldNumber, false), input: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.Normalize(tt.input)
			if err != nil {
				t.Fatalf("Normalize(%#v) = error %v, want %#v", tt.input, err, tt.want)
			}
			if got != tt.want {
				t.Errorf("Normalize(%#v) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeRejects(t *testing.T) {
	tests := []struct {
		name  string
		field UserField
		input any
	}{
		{name: "text in a number", field: field(FieldNumber, false), input: "many"},
		{name: "a word in a bool", field: field(FieldBool, false), input: "maybe"},
		{name: "an address with no at sign", field: field(FieldEmail, false), input: "nobody"},
		{name: "a date that is not one", field: field(FieldDate, false), input: "last tuesday"},
		{name: "a missing required value", field: field(FieldText, true), input: nil},
		{name: "an empty required value", field: field(FieldText, true), input: "   "},
		{name: "an empty required number", field: field(FieldNumber, true), input: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.field.Normalize(tt.input); err == nil {
				t.Errorf("Normalize(%#v) accepted the value, want an error", tt.input)
			}
		})
	}
}

// The error names the field, because it is shown to whoever is editing.
func TestNormalizeErrorNamesTheField(t *testing.T) {
	_, err := UserField{Name: "phone_verified", Type: FieldBool}.Normalize("maybe")
	if err == nil {
		t.Fatal("want an error")
	}

	if got := err.Error(); got != "phone_verified: must be true or false" {
		t.Errorf("error = %q", got)
	}
}

func TestFieldTypeValid(t *testing.T) {
	for _, known := range FieldTypes {
		if !known.Valid() {
			t.Errorf("%s.Valid() = false, want true", known)
		}
	}

	for _, unknown := range []FieldType{"", "blob", "Text"} {
		if unknown.Valid() {
			t.Errorf("%q.Valid() = true, want false", unknown)
		}
	}
}

// bound is a pointer to a number, which is how a field carries a bound it
// might not have.
//
//go:fix inline
func bound(value float64) *float64 {
	return new(value)
}

// TestNormalizeRules covers the rules a field can put on its values: the
// checks that turn a field definition into something an admin can rely on.
func TestNormalizeRules(t *testing.T) {
	tests := []struct {
		name    string
		field   UserField
		input   any
		wantErr bool
	}{
		{
			name:  "a number inside its bounds",
			field: UserField{Name: "age", Type: FieldNumber, Min: bound(18), Max: bound(120)},
			input: "30",
		},
		{
			name:    "a number below its smallest",
			field:   UserField{Name: "age", Type: FieldNumber, Min: bound(18)},
			input:   "17",
			wantErr: true,
		},
		{
			name:    "a number above its largest",
			field:   UserField{Name: "age", Type: FieldNumber, Max: bound(120)},
			input:   "121",
			wantErr: true,
		},
		{
			name:  "text of an allowed length",
			field: UserField{Name: "nickname", Type: FieldText, Min: bound(2), Max: bound(8)},
			input: "Mira",
		},
		{
			name:    "text that is too short",
			field:   UserField{Name: "nickname", Type: FieldText, Min: bound(2)},
			input:   "M",
			wantErr: true,
		},
		{
			name:    "text that is too long",
			field:   UserField{Name: "nickname", Type: FieldText, Max: bound(3)},
			input:   "Mirabel",
			wantErr: true,
		},
		{
			name:  "text with the prefix it needs",
			field: UserField{Name: "phone", Type: FieldText, StartsWith: "+"},
			input: "+996700000001",
		},
		{
			name:    "text without the prefix it needs",
			field:   UserField{Name: "phone", Type: FieldText, StartsWith: "+"},
			input:   "996700000001",
			wantErr: true,
		},
		{
			name:  "an empty optional field skips its rules",
			field: UserField{Name: "phone", Type: FieldText, StartsWith: "+", Min: bound(5)},
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.field.Normalize(tt.input)

			if tt.wantErr && err == nil {
				t.Fatalf("Normalize(%#v) = no error, want one", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Normalize(%#v) = error %v, want none", tt.input, err)
			}
		})
	}
}

// TestFieldValidate covers the definition itself: a rule a field cannot keep
// is a mistake in the panel, and is refused there rather than confusing
// whoever edits a record later.
func TestFieldValidate(t *testing.T) {
	tests := []struct {
		name    string
		field   UserField
		wantErr bool
	}{
		{name: "text with a length", field: UserField{Name: "a", Type: FieldText, Max: bound(10)}},
		{name: "a number with bounds", field: UserField{Name: "a", Type: FieldNumber, Min: bound(0), Max: bound(9)}},
		{name: "a plain bool", field: UserField{Name: "a", Type: FieldBool}},
		{
			name:    "a bool cannot be bounded",
			field:   UserField{Name: "a", Type: FieldBool, Max: bound(10)},
			wantErr: true,
		},
		{
			name:    "a date cannot have a prefix",
			field:   UserField{Name: "a", Type: FieldDate, StartsWith: "2026"},
			wantErr: true,
		},
		{
			name:    "a largest below its smallest",
			field:   UserField{Name: "a", Type: FieldNumber, Min: bound(10), Max: bound(1)},
			wantErr: true,
		},
		{
			name:    "a negative length",
			field:   UserField{Name: "a", Type: FieldText, Min: bound(-1)},
			wantErr: true,
		},
		{
			name:    "an unknown type",
			field:   UserField{Name: "a", Type: "blob"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("Validate() = no error, want one")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() = %v, want none", err)
			}
		})
	}
}

// TestBuiltinFieldsAreDescribed checks the list the panel draws its table
// from: every built-in field says what it is, and none of them claims to be
// something that could be stored in user_fields.
func TestBuiltinFieldsAreDescribed(t *testing.T) {
	seen := map[string]bool{}

	for _, field := range BuiltinFields() {
		if field.Name == "" || field.Label == "" {
			t.Errorf("field %+v is missing a name or a label", field)
		}

		if !field.Type.Valid() {
			t.Errorf("%s has type %q, which is not a field type", field.Name, field.Type)
		}

		if !field.IsBuiltin {
			t.Errorf("%s is in BuiltinFields but is not marked as built in", field.Name)
		}

		if err := field.Validate(); err != nil {
			t.Errorf("%s carries a rule it cannot keep: %v", field.Name, err)
		}

		if seen[field.Name] {
			t.Errorf("%s is listed twice", field.Name)
		}
		seen[field.Name] = true
	}
}

// The columns of the record and the names of the built-in fields have to be
// the same set: the panel reads a value by the field's name, so a name with
// no column behind it would show nothing and save nowhere.
func TestBuiltinFieldsMatchTheColumns(t *testing.T) {
	columns := map[string]bool{
		FieldEmailName:           true,
		FieldIsEmailVerifiedName: true,
		FieldFirstNameName:       true,
		FieldLastNameName:        true,
		FieldIsActiveName:        true,

		FieldIsPasswordTemporaryName: true,
	}

	for _, field := range BuiltinFields() {
		if !columns[field.Name] {
			t.Errorf("%s is described but is not one of the record's columns", field.Name)
		}
		delete(columns, field.Name)
	}

	for name := range columns {
		t.Errorf("%s is a column of the record but is not described", name)
	}
}

func TestIsBuiltinField(t *testing.T) {
	for _, name := range []string{"email", "first_name", "is_active"} {
		if !IsBuiltinField(name) {
			t.Errorf("IsBuiltinField(%q) = false, want true", name)
		}
	}

	for _, name := range []string{"", "nickname", "Email", "is_active2"} {
		if IsBuiltinField(name) {
			t.Errorf("IsBuiltinField(%q) = true, want false", name)
		}
	}
}

// FullName is what the panel calls a user when it has room for one line.
func TestUserFullName(t *testing.T) {
	tests := []struct {
		name string
		user User
		want string
	}{
		{name: "both names", user: User{FirstName: "Mira", LastName: "Testova"}, want: "Mira Testova"},
		{name: "first only", user: User{FirstName: "Mira"}, want: "Mira"},
		{name: "last only", user: User{LastName: "Testova"}, want: "Testova"},
		{name: "neither", user: User{Email: "mira@example.com"}, want: "mira@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.FullName(); got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// SetPassword keeps a hash that checks against the password and nothing else,
// and refuses a password bcrypt would not read to the end.
func TestUserSetPassword(t *testing.T) {
	var user User

	if err := user.SetPassword("long-enough"); err != nil {
		t.Fatalf("SetPassword() = %v", err)
	}

	if !user.HasPassword || user.PasswordHash == "" || user.PasswordHash == "long-enough" {
		t.Fatalf("hash = %q, has password = %v; want a hash", user.PasswordHash, user.HasPassword)
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("long-enough")) != nil {
		t.Error("the hash does not match the password it was made from")
	}

	if err := user.SetPassword(strings.Repeat("p", 73)); !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("SetPassword(73 bytes) = %v, want ErrPasswordTooLong", err)
	}
}
