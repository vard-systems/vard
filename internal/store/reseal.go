package store

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/jose"
	"vard/internal/model"
)

// lockForUpdate holds a row while its value is sealed again, so a write in
// between is not lost.
var lockForUpdate = clause.Locking{Strength: "UPDATE"}

// Reseal opens a sealed value kept where it says and seals it again under
// the current key; changed is false when it was current already.
type Reseal func(where jose.Context, sealed []byte) (resealed []byte, changed bool, err error)

// sealedColumn is one column that keeps sealed values: how each row's value
// is read and written, and where it is bound to.
type sealedColumn struct {
	table, column string
	// key is the column a row is known by.
	key string
	// encoding is how the column holds the bytes: "" raw, or base64.
	encoding *base64.Encoding
	where    func(key string) jose.Context
}

func byID(where func(id uuid.UUID) jose.Context) func(string) jose.Context {
	return func(key string) jose.Context {
		id, _ := uuid.Parse(key)
		return where(id)
	}
}

// sealedColumns are every column that keeps a sealed value. A secret added
// to a model goes here too, or `secrets reseal` leaves it under the old key.
var sealedColumns = []sealedColumn{
	{"server_secrets", "value", "name", nil, func(name string) jose.Context {
		return model.ServerSecret{Name: name}.ValueWhere()
	}},
	{"signing_keys", "private_key", "kid", nil, func(kid string) jose.Context {
		return model.SigningKey{KID: kid}.PrivateKeyWhere()
	}},
	{"mfa_factors", "secret", "id", base64.StdEncoding, byID(func(id uuid.UUID) jose.Context {
		return model.MFA{Base: model.Base{ID: id}}.SecretWhere()
	})},
	{"user_factors", "secret", "id", base64.StdEncoding, byID(func(id uuid.UUID) jose.Context {
		return model.UserFactor{Base: model.Base{ID: id}}.SecretWhere()
	})},
	{"social_providers", "client_secret", "id", nil, byID(func(id uuid.UUID) jose.Context {
		return model.SocialProvider{Base: model.Base{ID: id}}.ClientSecretWhere()
	})},
	{"social_providers", "private_key", "id", nil, byID(func(id uuid.UUID) jose.Context {
		return model.SocialProvider{Base: model.Base{ID: id}}.PrivateKeyWhere()
	})},
	{"sso_connections", "client_secret", "id", nil, byID(func(id uuid.UUID) jose.Context {
		return model.SSOConnection{Base: model.Base{ID: id}}.ClientSecretWhere()
	})},
	{"sso_connections", "sp_key", "id", nil, byID(func(id uuid.UUID) jose.Context {
		return model.SSOConnection{Base: model.Base{ID: id}}.SPKeyWhere()
	})},
	{"mail_settings", "password", "id", nil, byID(func(id uuid.UUID) jose.Context {
		return model.MailSettings{Base: model.Base{ID: id}}.PasswordWhere()
	})},
	{"refresh_tokens", "successor_sealed", "id", nil, byID(func(id uuid.UUID) jose.Context {
		return model.RefreshToken{Base: model.Base{ID: id}}.SuccessorWhere()
	})},
	{"sso_logins", "saml_response", "relay_hash", base64.RawStdEncoding, model.SAMLResponseWhere},
}

// ResealAll seals every sealed value again under the current key, column by
// column, each row in a transaction of its own, and says how many it changed.
func (s *Store) ResealAll(ctx context.Context, reseal Reseal) (int, error) {
	changed := 0

	for _, column := range sealedColumns {
		n, err := s.resealColumn(ctx, column, reseal)
		changed += n
		if err != nil {
			return changed, fmt.Errorf("%s.%s: %w", column.table, column.column, err)
		}
	}

	return changed, nil
}

func (s *Store) resealColumn(ctx context.Context, column sealedColumn, reseal Reseal) (int, error) {
	var keys []string
	err := s.db.WithContext(ctx).Table(column.table).
		Where(fmt.Sprintf("%s IS NOT NULL AND length(%s) > 0", column.column, column.column)).
		Pluck(column.key, &keys).Error
	if err != nil {
		return 0, err
	}

	changed := 0
	for _, key := range keys {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var raw []byte
			row := tx.Table(column.table).Where(column.key+" = ?", key).
				Select(column.column).Clauses(lockForUpdate).Row()
			if err := row.Scan(&raw); err != nil {
				return err
			}

			stored := raw
			if column.encoding != nil {
				decoded, err := column.encoding.DecodeString(string(raw))
				if err != nil {
					return fmt.Errorf("row %s is not the encoding it should be: %w", key, err)
				}
				stored = decoded
			}

			resealed, moved, err := reseal(column.where(key), stored)
			if err != nil || !moved {
				return err
			}

			var value any = resealed
			if column.encoding != nil {
				value = column.encoding.EncodeToString(resealed)
			}
			if err := tx.Table(column.table).Where(column.key+" = ?", key).Update(column.column, value).Error; err != nil {
				return err
			}

			changed++
			return nil
		})
		if err != nil {
			return changed, fmt.Errorf("row %s: %w", key, err)
		}
	}

	return changed, nil
}
