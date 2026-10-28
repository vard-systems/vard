package sso

import (
	"testing"

	"vard/internal/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	// The model sends its certificate as sp_certificate too; the panel reads
	// the copy under service_provider.
	tstypes.Match(t, "SSOConnection", connectionResponse{}, "sp_certificate")
	tstypes.Match(t, "SSOIdentityProvider", identityProvider{})
	tstypes.Match(t, "SSOTestResult", testResponse{})
	tstypes.Match(t, "SSODomainProof", domainProof{})
}
