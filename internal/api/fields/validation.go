package fields

import (
	"net/http"
	"regexp"
	"strings"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
)

var (
	nameTaken   = respond.Define(http.StatusConflict, "user_field_name_taken", respond.Admin)
	fieldAbsent = respond.Define(http.StatusNotFound, "user_field_not_found", respond.Admin)
)

// column is what a field may be called: the same shape as a column name,
// because that is what it stands in for.
var column = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func init() {
	validate.Register("column", column.MatchString)

	validate.Register("fieldtype", func(value string) bool {
		return model.FieldType(value).Valid()
	})
}

func (r *fieldRequest) newField() (*model.UserField, error) {
	r.Name = strings.ToLower(strings.TrimSpace(r.Name))
	r.Type = strings.TrimSpace(r.Type)

	if err := validate.Struct(r); err != nil {
		return nil, err
	}

	if model.IsBuiltinField(r.Name) {
		return nil, respond.Fault{
			Status: http.StatusConflict,
			Message: r.Name + " is a built-in field: every record has one already, " +
				"and two fields with one name would be two places to look",
		}
	}

	field := &model.UserField{Name: r.Name, Type: model.FieldType(r.Type)}
	if err := r.rulesRequest.applyTo(field); err != nil {
		return nil, err
	}

	return field, nil
}

func (r *rulesRequest) applyTo(field *model.UserField) error {
	r.Label = strings.TrimSpace(r.Label)
	r.StartsWith = strings.TrimSpace(r.StartsWith)

	if err := validate.Struct(r); err != nil {
		return err
	}

	label := r.Label
	if label == "" {
		label = field.Name
	}

	field.Label = label
	field.IsRequired = r.IsRequired
	field.IsUnique = r.IsUnique
	field.Min = r.Min
	field.Max = r.Max
	field.StartsWith = r.StartsWith

	if err := field.Validate(); err != nil {
		return respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
	}

	return nil
}
