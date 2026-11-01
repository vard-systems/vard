package applications

import (
	"testing"

	"vard/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "Application", applicationResponse{})
	tstypes.Match(t, "ApplicationWithSecret", secretResponse{})
	tstypes.Match(t, "ApplicationPage", pageResponse{})
	tstypes.Match(t, "APIAccess", accessAPI{})
}
