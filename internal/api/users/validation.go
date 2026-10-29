package users

import (
	"context"
	"net/http"
	"unicode/utf8"

	"github.com/google/uuid"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
	"vard/internal/store"
)

var (
	identityAbsent   = respond.Define(http.StatusNotFound, "user_social_account_not_found", respond.Admin)
	noRoles          = respond.Define(http.StatusBadRequest, "user_roles_missing", respond.Admin)
	roleUnknown      = respond.Define(http.StatusBadRequest, "user_role_unknown", respond.Admin)
	roleNotHeld      = respond.Define(http.StatusNotFound, "user_role_not_held", respond.Admin)
	emailTaken       = respond.Define(http.StatusConflict, "user_email_taken", respond.Admin)
	passwordRequired = respond.Define(http.StatusBadRequest, "user_password_required", respond.Admin)
	passwordTooShort = respond.Define(http.StatusBadRequest, "user_password_too_short", respond.Admin)
	passwordBreached = respond.Define(http.StatusBadRequest, "user_password_breached", respond.Admin)
)

const minPasswordLength = model.MinUserPasswordLength

func (r *userRequest) validate(creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	switch {
	case r.Password == "" && creating:
		return passwordRequired.With()
	case r.Password != "" && utf8.RuneCountInString(r.Password) < minPasswordLength:
		return passwordTooShort.With("min", minPasswordLength)
	}

	return nil
}

func normalize(
	ctx context.Context,
	st *store.Store,
	fields []model.UserField,
	values map[string]any,
	self uuid.UUID,
) (map[string]any, error) {
	data := make(map[string]any, len(fields))

	for _, field := range fields {
		value, err := field.Normalize(values[field.Name])
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
