package mailsettings

import (
	"testing"

	"vard/internal/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "MailSettings", settingsResponse{})
	tstypes.Match(t, "MailResponse", response{})
	tstypes.Match(t, "MailLanguageContent", languageContent{})
	tstypes.Match(t, "MailContent", contentResponse{})
}
