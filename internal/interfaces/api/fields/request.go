package fields

const targetType = "user_field"

// rulesRequest is a field's rules; name and type cannot change.
type rulesRequest struct {
	Label      string   `json:"label" validate:"max=100"`
	IsRequired bool     `json:"is_required"`
	IsUnique   bool     `json:"is_unique"`
	Min        *float64 `json:"min"`
	Max        *float64 `json:"max"`
	StartsWith string   `json:"starts_with" validate:"max=64"`
}

// fieldRequest is what a create sends: the same rules, plus what the field is.
type fieldRequest struct {
	rulesRequest

	Name string `json:"name" validate:"required,column"`
	Type string `json:"type" validate:"required,fieldtype"`
}
