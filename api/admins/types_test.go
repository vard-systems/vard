package admins

import (
	"testing"

	"vard/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "AdminRecord", adminResponse{})
	tstypes.Match(t, "AdminAssignment", assignmentResponse{})
	tstypes.Match(t, "AdminPage", pageResponse{})
	tstypes.Match(t, "Reference", reference{})
}
