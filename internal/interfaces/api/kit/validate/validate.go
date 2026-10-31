// Package validate checks request bodies against their struct tags and turns
// what fails into problems.
package validate

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"

	"vard/internal/interfaces/api/kit/respond"
)

// instance is shared: building one is expensive, and it is safe to use from
// every request at once.
var instance = newValidator()

var rules = map[string]respond.Problem{
	"required":   respond.Define(http.StatusBadRequest, "validation.required", respond.Both),
	"email":      respond.Define(http.StatusBadRequest, "validation.email", respond.Both),
	"max":        respond.Define(http.StatusBadRequest, "validation.max", respond.Both),
	"min":        respond.Define(http.StatusBadRequest, "validation.min", respond.Both),
	"oneof":      respond.Define(http.StatusBadRequest, "validation.oneof", respond.Both),
	"startswith": respond.Define(http.StatusBadRequest, "validation.startswith", respond.Both),
	"eqfield":    respond.Define(http.StatusBadRequest, "validation.eqfield", respond.Both),
}

var invalid = respond.Define(http.StatusBadRequest, "validation.invalid", respond.Both)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return field.Name
		}

		return name
	})

	return v
}

func Register(tag string, valid func(value string) bool) {
	if err := instance.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return valid(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("validate: register %q: %v", tag, err))
	}

	rules[tag] = respond.Define(http.StatusBadRequest, "validation."+tag, respond.Admin)
}

func Struct(v any) error {
	err := instance.Struct(v)
	if err == nil {
		return nil
	}

	var broken validator.ValidationErrors
	if !errors.As(err, &broken) {
		return err
	}

	return Fault(broken[0])
}

func Fault(broken validator.FieldError) respond.Fault {
	problem, ok := rules[broken.Tag()]
	if !ok {
		problem = invalid
	}

	params := map[string]any{"field": broken.Field()}

	if param := broken.Param(); param != "" {
		switch broken.Tag() {
		case "max", "min":
			params[broken.Tag()] = param
		case "oneof":
			params["values"] = strings.ReplaceAll(param, " ", ", ")
		case "startswith":
			params["prefix"] = param
		case "eqfield":
			params["other"] = toSnake(param)
		}
	}

	return problem.Fault(params)
}

func toSnake(name string) string {
	var b strings.Builder

	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}

	return b.String()
}

// These read one PATCH field: the value sent, or `current` when it was not.
// Requests carry pointers because a plain value cannot tell false, "" or 0 from
// "not sent".

func Flag(sent *bool, current bool) bool {
	if sent == nil {
		return current
	}

	return *sent
}

func Text(sent *string, current string) string {
	if sent == nil {
		return current
	}

	return strings.TrimSpace(*sent)
}

func Lower(sent *string, current string) string {
	return strings.ToLower(Text(sent, current))
}

func Number(sent *int, current int) int {
	if sent == nil {
		return current
	}

	return *sent
}
