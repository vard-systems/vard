package model

import (
	"time"

	"github.com/google/uuid"
)

// PushedRequestLifetime is how long a pushed authorization request waits for
// the browser (RFC 9126 recommends well under ten minutes).
const PushedRequestLifetime = 90 * time.Second

// RequestURIPrefix starts every request_uri this server hands out.
const RequestURIPrefix = "urn:ietf:params:oauth:request_uri:"

// PushedRequest is an authorization request a client sent server to server
// (RFC 9126), waiting for the browser that will carry its request_uri. Used
// once.
type PushedRequest struct {
	Base

	HandleHash    string       `gorm:"size:64;uniqueIndex;not null"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`
	// Params are the request's parameters, as the client pushed them.
	Params string `gorm:"type:text;not null"`

	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (PushedRequest) TableName() string {
	return "pushed_requests"
}
