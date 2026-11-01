package otp

import (
	"testing"

	"vard/internal/interfaces/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "OTPSettings", settingsResponse{})
	tstypes.Match(t, "OTPFlow", usingFlow{})
	tstypes.Match(t, "OTPLimits", limits{})
	tstypes.Match(t, "OTPResponse", response{})
}
