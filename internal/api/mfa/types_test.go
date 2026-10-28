package mfa

import (
	"testing"

	"vard/internal/api/kit/tstypes"
	"vard/internal/auth"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "MfaStatus", auth.MFAStatus{})
	tstypes.Match(t, "MfaEnrolment", auth.Enrolment{})
	tstypes.Match(t, "AdminPasskey", auth.Passkey{})
}
