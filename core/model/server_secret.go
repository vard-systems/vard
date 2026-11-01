package model

import (
	"time"

	"vard/tools/jose"
)

// ServerSecret is a secret the installation makes for itself once and keeps
// sealed: the key its keyed hashes are made with (ServerSecretTag).
type ServerSecret struct {
	Name      string    `gorm:"size:32;primaryKey"`
	Value     []byte    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (ServerSecret) TableName() string {
	return "server_secrets"
}

// ServerSecretTag keys recovery codes, device user codes, sign-in throttle
// keys and known-device cookies.
const ServerSecretTag = "tag"

func (s ServerSecret) ValueWhere() jose.Context {
	return jose.Bind("server_secrets", "value", s.Name)
}
