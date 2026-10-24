package adminauth

import (
	"testing"

	"loginer/internal/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "Admin", adminResponse{})
	tstypes.Match(t, "AdminSession", sessionResponse{})
}
