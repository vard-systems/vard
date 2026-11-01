package apis

import (
	"testing"

	"vard/internal/interfaces/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "API", apiResponse{})
	tstypes.Match(t, "APIScope", scopeResponse{})
	tstypes.Match(t, "APIApplication", applicationAccess{})
	tstypes.Match(t, "APIIntrospector", introspector{})
	tstypes.Match(t, "APILogEntry", logEntry{})
}
