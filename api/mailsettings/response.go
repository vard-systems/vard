package mailsettings

import (
	"net/http"

	"vard/api/kit/respond"
	"vard/core/model"
)

// The problems these endpoints answer with.
var (
	testNeedsRecipient = respond.Define(http.StatusBadRequest, "mail_test_needs_recipient", respond.Admin)

	testFailed = respond.Define(http.StatusBadGateway, "mail_test_failed", respond.Admin)

	contentKeyUnknown = respond.Define(http.StatusBadRequest, "mail_content_key_unknown", respond.Admin)
	contentTooLong    = respond.Define(http.StatusBadRequest, "mail_content_too_long", respond.Admin)
)

// settingsResponse is built by hand so the password can never leave; only
// whether one exists is said.
type settingsResponse struct {
	IsEnabled   bool   `json:"is_enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Encryption  string `json:"encryption"`
	Username    string `json:"username"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`

	// HasPassword says a password is stored, without saying what it is. The
	// form shows "set" rather than asking for one that is already there.
	HasPassword bool `json:"has_password"`
}

// response includes the encryption options so the panel's picker comes from the
// server.
type response struct {
	Mail        settingsResponse       `json:"mail"`
	Encryptions []model.MailEncryption `json:"encryptions"`
}

func newResponse(settings model.MailSettings) response {
	return response{
		Mail: settingsResponse{
			IsEnabled:   settings.IsEnabled,
			Host:        settings.Host,
			Port:        settings.Port,
			Encryption:  string(settings.Encryption),
			Username:    settings.Username,
			FromAddress: settings.FromAddress,
			FromName:    settings.FromName,
			HasPassword: len(settings.Password) > 0,
		},
		Encryptions: model.MailEncryptions,
	}
}

// languageContent is one language's words for the emails: what it says, and
// what English says, key by key.
type languageContent struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`

	// Offered is whether the sign-in pages are shown in this language. One
	// that is not still has its text, and the panel marks it.
	Offered bool `json:"offered"`

	// Messages is what this language says, for the email keys alone, and
	// only where it says something: a key it is missing falls back to Base.
	Messages map[string]string `json:"messages"`

	// Base is the shipped English for the same keys, which is what an email
	// goes out in where a language has nothing.
	Base map[string]string `json:"base"`
}

func newLanguageContent(language model.Language, messages, english map[string]string) languageContent {
	own := map[string]string{}
	base := map[string]string{}

	for _, key := range model.MailMessageKeys() {
		if text, ok := messages[key]; ok {
			own[key] = text
		}
		base[key] = english[key]
	}

	return languageContent{
		Code:       language.Code,
		Name:       language.Name,
		NativeName: language.NativeName,
		Offered:    language.IsEnabled,
		Messages:   own,
		Base:       base,
	}
}

// contentResponse is every email the server sends and every language it can
// be sent in: the catalog once, and the text per language.
type contentResponse struct {
	Messages  []model.MailMessageSpec `json:"messages"`
	Languages []languageContent       `json:"languages"`
}
