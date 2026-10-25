// Package respond is how every handler answers: the catalogue of problems, and
// the functions that send one or log an internal failure.
package respond

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/internal/store"
)

type Fault struct {
	Status  int
	Code    string
	Params  map[string]any
	Message string
}

func (f Fault) Error() string {
	return f.Message
}

func Fail(c *gin.Context, problem Problem, pairs ...any) {
	Write(c, problem.With(pairs...))
}

func Abort(c *gin.Context, problem Problem, pairs ...any) {
	Write(c, problem.With(pairs...))
	c.Abort()
}

func Write(c *gin.Context, fault Fault) {
	body := gin.H{"error": fault.Message}
	if fault.Code != "" {
		body["code"] = fault.Code
	}
	if len(fault.Params) > 0 {
		body["params"] = fault.Params
	}

	c.JSON(fault.Status, body)
}

// Invalid passes on the sentence a model's own validation refused a value
// with, which names the field and the rule; everything else is a Problem.
func Invalid(message string) error {
	return Fault{Status: http.StatusBadRequest, Message: message}
}

var Forbidden = NotAllowed.Fault(nil)

// Failure passes a Fault through and logs anything else with `note`, answering
// with the internal problem, so database errors never reach a browser.
func Failure(c *gin.Context, log *slog.Logger, err error, note string) {
	if fault, ok := errors.AsType[Fault](err); ok {
		Write(c, fault)
		return
	}

	log.Error(note, "error", err)
	Write(c, Internal.Fault(nil))
}

// Taken answers a failed write: with `taken` when the row clashed with a
// unique value already in use (a name, an email, an identifier), and through
// Failure otherwise.
func Taken(c *gin.Context, log *slog.Logger, err error, taken Problem, note string) {
	if errors.Is(err, store.ErrDuplicate) {
		Fail(c, taken)
		return
	}

	Failure(c, log, err, note)
}

// Image sends a stored picture. Asked for by its current version, it is the
// same bytes forever, so a browser keeps it (`private` behind a session, so no
// shared cache does); asked for by any other, it is checked every time, and a
// browser that already has it is told so without the bytes. The nosniff
// header the API sends on everything keeps it an image.
func Image(c *gin.Context, contentType string, data []byte, version string, private bool) {
	cacheControl := "no-cache"
	if c.Query("v") == version {
		cacheControl = "public, max-age=31536000, immutable"
		if private {
			cacheControl = "private, max-age=31536000, immutable"
		}
	}

	etag := `"` + version + `"`
	c.Header("Cache-Control", cacheControl)
	c.Header("ETag", etag)

	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	c.Data(http.StatusOK, contentType, data)
}
