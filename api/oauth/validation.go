package oauth

import (
	"net/http"
	"strings"

	"vard/internal/application/oidc"
)

func invalidRequest(description string) error {
	return &oidc.Error{Code: oidc.ErrInvalidRequest, Description: description, Status: http.StatusBadRequest}
}

// quoteSafe keeps a description inside the quoted string of a
// WWW-Authenticate header: RFC 6750 allows neither quotes nor backslashes in
// it.
func quoteSafe(description string) string {
	return strings.NewReplacer(`"`, "'", `\`, "/").Replace(description)
}
