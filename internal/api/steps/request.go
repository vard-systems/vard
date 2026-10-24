package steps

// stepRequest is the answer to the Agreements or Consent page: the sign-in it
// is about, and yes or no.
type stepRequest struct {
	Request string `json:"request" validate:"required,max=64"`
	Yes     bool   `json:"yes"`
}

// deviceRequest is the person's answer for a device, by the code it showed.
type deviceRequest struct {
	UserCode string `json:"user_code" validate:"required,max=16"`
	Yes      bool   `json:"yes"`
}
