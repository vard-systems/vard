package mfa

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

// codeRequest is a code from the authenticator app, or a recovery code.
type codeRequest struct {
	Code string `json:"code" binding:"required,max=32"`
}

// optionalCodeRequest is the body of starting a set-up: a code from the
// current authenticator when one is being replaced, and nothing otherwise.
type optionalCodeRequest struct {
	Code string `json:"code" binding:"max=32"`
}

func bindOptional(c *gin.Context, into *optionalCodeRequest) error {
	if c.Request.ContentLength == 0 {
		return nil
	}
	return c.ShouldBindJSON(into)
}

// passkeyRequest is the browser's answer to navigator.credentials.create,
// and what the administrator calls the passkey.
type passkeyRequest struct {
	Label    string          `json:"label" binding:"max=100"`
	Response json.RawMessage `json:"response" binding:"required"`
}
