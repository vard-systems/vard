package model

import "slices"

// How a person proved who they are, as an ID token's amr claim says it (RFC
// 8176). A session keeps the methods its sign-in used; mfa is worked out from
// them.
const (
	AMRPassword     = "pwd"
	AMROneTimeCode  = "otp"
	AMRHardwareKey  = "hwk"
	AMRUserPresence = "user"
	AMRMultiFactor  = "mfa"
)

// amrKind is the kind of factor each method is: something known, something
// held, or the person verified on the device that holds the key.
var amrKind = map[string]string{
	AMRPassword:     "knowledge",
	AMROneTimeCode:  "possession",
	AMRHardwareKey:  "possession",
	AMRUserPresence: "verification",
}

// AMR is the amr claim for a sign-in's methods: each method once, in the
// order used, and mfa when more than one kind of factor was proved. Nothing
// proved here (a sign-in through another provider alone) is no claim.
func AMR(methods []string) []string {
	out := []string{}
	kinds := map[string]bool{}
	for _, method := range methods {
		if kind, ok := amrKind[method]; ok && !slices.Contains(out, method) {
			out = append(out, method)
			kinds[kind] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	if len(kinds) > 1 {
		out = append(out, AMRMultiFactor)
	}
	return out
}
