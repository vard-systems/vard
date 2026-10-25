package oidc

// Mutual TLS (RFC 8705). A client reaches the token, PAR, revocation,
// introspection, device and userinfo endpoints on a host of its own
// (mtls_endpoint_aliases), whose TLS proxy asks for a certificate and passes
// it on. A client may authenticate with it — tls_client_auth, a certificate
// from a CA the server trusts carrying the client's registered subject, or
// self_signed_tls_client_auth, one of the certificates it registered — and
// an application may have its access tokens bound to it: they carry the
// certificate's thumbprint in cnf, and userinfo takes one only over a
// connection with that certificate.

// confirmationCertificate is the cnf member naming the certificate a token
// is bound to (RFC 8705 §3.1).
const confirmationCertificate = "x5t#S256"

// boundCertificate is the thumbprint of the certificate an access token is
// bound to, "" when it is not.
func boundCertificate(raw map[string]any) string {
	cnf, _ := raw["cnf"].(map[string]any)
	thumbprint, _ := cnf[confirmationCertificate].(string)
	return thumbprint
}
