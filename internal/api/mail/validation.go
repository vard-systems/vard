package mail

import (
	"net/http"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/jose"
	"loginer/internal/mail"
	"loginer/internal/model"
)

func (r *settingsRequest) applyTo(settings *model.MailSettings, sealer *jose.Sealer) ([]string, error) {
	updated := *settings
	updated.IsEnabled = validate.Flag(r.IsEnabled, settings.IsEnabled)
	updated.Host = validate.Lower(r.Host, settings.Host)
	updated.Port = validate.Number(r.Port, settings.Port)
	updated.Encryption = model.MailEncryption(validate.Lower(r.Encryption, string(settings.Encryption)))
	updated.Username = validate.Text(r.Username, settings.Username)
	updated.FromAddress = validate.Lower(r.FromAddress, settings.FromAddress)
	updated.FromName = validate.Text(r.FromName, settings.FromName)

	// A password is write-only: the request sends nothing, a new one, or "" for
	// none.
	passwordChanged := false
	if r.Password != nil {
		password := *r.Password

		switch {
		case password == "":
			passwordChanged = len(settings.Password) > 0
			updated.Password = nil
		default:
			sealed, err := sealer.SealBytes([]byte(password))
			if err != nil {
				return nil, err
			}
			passwordChanged = true
			updated.Password = sealed
		}
	}

	if err := updated.Validate(); err != nil {
		return nil, respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
	}

	changed := []string{}
	for _, field := range []struct {
		name  string
		moved bool
	}{
		{"is_enabled", updated.IsEnabled != settings.IsEnabled},
		{"host", updated.Host != settings.Host},
		{"port", updated.Port != settings.Port},
		{"encryption", updated.Encryption != settings.Encryption},
		{"username", updated.Username != settings.Username},
		{"password", passwordChanged},
		{"from_address", updated.FromAddress != settings.FromAddress},
		{"from_name", updated.FromName != settings.FromName},
	} {
		if field.moved {
			changed = append(changed, field.name)
		}
	}

	*settings = updated

	return changed, nil
}

func (r *testRequest) settings(stored *model.MailSettings, sealer *jose.Sealer) (mail.Settings, error) {
	trying := *stored

	trying.IsEnabled = true
	trying.Host = validate.Lower(r.Host, stored.Host)
	trying.Port = validate.Number(r.Port, stored.Port)
	trying.Encryption = model.MailEncryption(validate.Lower(r.Encryption, string(stored.Encryption)))
	trying.Username = validate.Text(r.Username, stored.Username)
	trying.FromAddress = validate.Lower(r.FromAddress, stored.FromAddress)
	trying.FromName = validate.Text(r.FromName, stored.FromName)

	if err := trying.Validate(); err != nil {
		return mail.Settings{}, respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
	}

	if r.Password != nil {
		return mail.Of(trying, *r.Password), nil
	}

	password := ""
	if len(stored.Password) > 0 {
		plain, err := sealer.OpenBytes(stored.Password)
		if err != nil {
			return mail.Settings{}, err
		}
		password = string(plain)
	}

	return mail.Of(trying, password), nil
}

func (r *contentRequest) validate() error {
	for key, text := range r.Messages {
		if !model.IsMailMessageKey(key) {
			return contentKeyUnknown.With("key", key)
		}

		if len(text) > model.MaxMessageLength {
			return respond.Fault{
				Status:  http.StatusBadRequest,
				Message: key + " is longer than a message may be",
			}
		}
	}

	return nil
}
