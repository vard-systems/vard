package model

import "testing"

func TestStatusValid(t *testing.T) {
	valid := []Status{StatusInvited, StatusActive, StatusSuspended, StatusDisabled}
	for _, status := range valid {
		if !status.Valid() {
			t.Errorf("%s.Valid() = false, want true", status)
		}
	}

	invalid := []Status{"", "Active", "deleted"}
	for _, status := range invalid {
		if status.Valid() {
			t.Errorf("%q.Valid() = true, want false", status)
		}
	}
}

func TestMFAMethodValid(t *testing.T) {
	if !MFAMethodTOTP.Valid() || !MFAMethodWebAuthn.Valid() {
		t.Error("a known method reported itself invalid")
	}
	if MFAMethod("sms").Valid() {
		t.Error(`MFAMethod("sms").Valid() = true, want false`)
	}
}
