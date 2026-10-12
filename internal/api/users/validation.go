package users

import (
	"context"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/google/uuid"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
	"loginer/internal/store"
)

const minPasswordLength = 8

func (r *userRequest) validate(creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	switch {
	case r.Password == "" && creating:
		return respond.Invalid("password is required")
	case r.Password != "" && utf8.RuneCountInString(r.Password) < minPasswordLength:
		return respond.Invalid(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	}

	return nil
}

func normalise(
	ctx context.Context,
	st *store.Store,
	fields []model.UserField,
	values map[string]any,
	self uuid.UUID,
) (map[string]any, error) {
	data := make(map[string]any, len(fields))

	for _, field := range fields {
		value, err := field.Normalise(values[field.Name])
		if err != nil {
			return nil, respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
		}

		if value == nil {
			continue
		}

		if field.IsUnique {
			taken, err := st.FieldValueTaken(ctx, field.Name, value, self)
			if err != nil {
				return nil, err
			}

			if taken {
				return nil, respond.Fault{
					Status:  http.StatusConflict,
					Message: field.Name + ": another user already has that value",
				}
			}
		}

		data[field.Name] = value
	}

	return data, nil
}
