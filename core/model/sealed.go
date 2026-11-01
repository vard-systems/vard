package model

import "vard/tools/jose"

// Where each secret the panel seals is kept, which the Sealer binds into it:
// sealed here, the value opens only here.

func (p SocialProvider) ClientSecretWhere() jose.Context {
	return jose.Bind("social_providers", "client_secret", p.ID)
}

func (p SocialProvider) PrivateKeyWhere() jose.Context {
	return jose.Bind("social_providers", "private_key", p.ID)
}

func (c SSOConnection) ClientSecretWhere() jose.Context {
	return jose.Bind("sso_connections", "client_secret", c.ID)
}

func (c SSOConnection) SPKeyWhere() jose.Context {
	return jose.Bind("sso_connections", "sp_key", c.ID)
}

func (m MailSettings) PasswordWhere() jose.Context {
	return jose.Bind("mail_settings", "password", m.ID)
}

// SuccessorWhere is where a replaced refresh token keeps its successor:
// bound to the replaced token's row, so a sealed value from anywhere else
// put there does not open as a refresh token.
func (t RefreshToken) SuccessorWhere() jose.Context {
	return jose.Bind("refresh_tokens", "successor_sealed", t.ID)
}

// SAMLResponseWhere is where a SAML answer is held for the sign-in its
// RelayState names, the key that row is found by when it is held.
func SAMLResponseWhere(relayHash string) jose.Context {
	return jose.Bind("sso_logins", "saml_response", relayHash)
}

func (f UserFactor) SecretWhere() jose.Context {
	return jose.Bind("user_factors", "secret", f.ID)
}

func (m MFA) SecretWhere() jose.Context {
	return jose.Bind("mfa_factors", "secret", m.ID)
}

func (k SigningKey) PrivateKeyWhere() jose.Context {
	return jose.Bind("signing_keys", "private_key", k.KID)
}
