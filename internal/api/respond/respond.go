package respond

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
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

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Invalid is BadRequest as an error, for validation that returns one.
func Invalid(message string) error {
	return Fault{Status: http.StatusBadRequest, Message: message}
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

var Forbidden = NotAllowed.Fault(nil)

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message)
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// Failure passes a Fault through and logs anything else with `note`, answering
// with the internal problem, so database errors never reach a browser.
func Failure(c *gin.Context, log *slog.Logger, err error, note string) {
	var fault Fault
	if errors.As(err, &fault) {
		Write(c, fault)
		return
	}

	log.Error(note, "error", err)
	Write(c, Internal.Fault(nil))
}
