package model

import "slices"

type Status string

const (
	StatusInvited   Status = "invited"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDisabled  Status = "disabled"
)

var Statuses = []Status{StatusInvited, StatusActive, StatusSuspended, StatusDisabled}

func (s Status) Valid() bool {
	return slices.Contains(Statuses, s)
}

func (s Status) String() string {
	return string(s)
}

type MFAMethod string

const (
	MFAMethodTOTP     MFAMethod = "totp"
	MFAMethodWebAuthn MFAMethod = "webauthn"
)

var MFAMethods = []MFAMethod{MFAMethodTOTP, MFAMethodWebAuthn}

func (m MFAMethod) Valid() bool {
	return slices.Contains(MFAMethods, m)
}

func (m MFAMethod) String() string {
	return string(m)
}
