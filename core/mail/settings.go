package mail

import (
	"context"
	"fmt"
	"net/mail"

	"vard/core/config"
	"vard/core/model"
	"vard/tools/jose"
)

type Store interface {
	MailSettings(ctx context.Context) (*model.MailSettings, error)
}

type Opener interface {
	OpenFor(where jose.Context, sealed []byte) ([]byte, error)
}

func FromStore(st Store, opener Opener) Source {
	return func(ctx context.Context) (Settings, error) {
		stored, err := st.MailSettings(ctx)
		if err != nil {
			return Settings{}, err
		}

		password := ""
		if len(stored.Password) > 0 {
			plain, err := opener.OpenFor(stored.PasswordWhere(), stored.Password)
			if err != nil {
				return Settings{}, fmt.Errorf("unseal the mail password: %w", err)
			}
			password = string(plain)
		}

		return Of(*stored, password), nil
	}
}

func Of(stored model.MailSettings, password string) Settings {
	return Settings{
		Enabled:     stored.IsEnabled,
		Host:        stored.Host,
		Port:        stored.Port,
		Encryption:  Encryption(stored.Encryption),
		Username:    stored.Username,
		Password:    password,
		FromAddress: stored.FromAddress,
		FromName:    stored.FromName,
	}
}

const implicitTLSPort = 465

func Seed(cfg config.Mail) (model.MailSettings, string) {
	settings := model.DefaultMailSettings()
	settings.IsEnabled = cfg.Host != ""
	settings.Host = cfg.Host
	settings.Username = cfg.Username

	if cfg.Port > 0 {
		settings.Port = cfg.Port
	}
	if settings.Port == implicitTLSPort {
		settings.Encryption = model.MailTLS
	}

	if parsed, err := mail.ParseAddress(cfg.From); err == nil {
		settings.FromAddress = parsed.Address
		settings.FromName = parsed.Name
	}

	return settings, cfg.Password
}
