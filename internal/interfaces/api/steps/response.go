package steps

// stepResponse is where the browser goes after the Agreements or Consent page.
type stepResponse struct {
	RedirectTo string `json:"redirect_to"`
}
