package users

import (
	"testing"

	"vard/internal/api/kit/tstypes"
	"vard/internal/model"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "UserRecord", model.User{})
	tstypes.Match(t, "UserPage", pageResponse{})
	tstypes.Match(t, "RoleMapping", mappedRole{})
}
