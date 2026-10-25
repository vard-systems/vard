package fields

import "vard/internal/model"

// fieldResponse is one field as the panel sees it. Built-in fields are columns
// with no id or timestamps, so this is built by hand rather than returning the
// model.
type fieldResponse struct {
	ID string `json:"id,omitempty"`

	Name  string          `json:"name"`
	Label string          `json:"label"`
	Type  model.FieldType `json:"type"`

	IsRequired bool     `json:"is_required"`
	IsUnique   bool     `json:"is_unique"`
	Min        *float64 `json:"min"`
	Max        *float64 `json:"max"`
	StartsWith string   `json:"starts_with"`

	Position  int  `json:"position"`
	IsBuiltin bool `json:"is_builtin"`
}

func newFieldResponse(field model.UserField) fieldResponse {
	out := fieldResponse{
		Name:       field.Name,
		Label:      field.Label,
		Type:       field.Type,
		IsRequired: field.IsRequired,
		IsUnique:   field.IsUnique,
		Min:        field.Min,
		Max:        field.Max,
		StartsWith: field.StartsWith,
		Position:   field.Position,
		IsBuiltin:  field.IsBuiltin,
	}

	if !field.IsBuiltin {
		out.ID = field.ID.String()
	}

	return out
}

// listResponse is every field of a user record, built-ins first and marked,
// plus the types a new field may have.
type listResponse struct {
	Fields []fieldResponse   `json:"fields"`
	Types  []model.FieldType `json:"types"`
}

func newListResponse(added []model.UserField) listResponse {
	builtin := model.BuiltinFields()
	fields := make([]fieldResponse, 0, len(builtin)+len(added))

	for _, field := range builtin {
		fields = append(fields, newFieldResponse(field))
	}

	for _, field := range added {
		fields = append(fields, newFieldResponse(field))
	}

	return listResponse{Fields: fields, Types: model.FieldTypes}
}
