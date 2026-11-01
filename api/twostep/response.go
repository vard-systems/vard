package twostep

// recoveryCodesResponse carries recovery codes the one time they are shown:
// with a user's first factor, or when they ask for new ones. Empty otherwise.
type recoveryCodesResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}
