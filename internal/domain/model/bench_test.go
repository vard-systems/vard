package model

import "testing"

// The per-request costs of the token endpoint that do not touch the
// database: hashing the presented secret, checking PKCE, and deciding what
// the token carries. A password hash is deliberately slow and is here to
// show how slow: it bounds how many sign-ins one core can take a second.

func BenchmarkHashSecret(b *testing.B) {
	secret, _, err := NewSecret()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		HashSecret(secret)
	}
}

func BenchmarkVerifyPKCE(b *testing.B) {
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if !VerifyPKCE(challenge, verifier) {
		b.Fatal("the RFC 7636 example does not verify")
	}

	b.ReportAllocs()
	for b.Loop() {
		VerifyPKCE(challenge, verifier)
	}
}

func BenchmarkEvaluateToken(b *testing.B) {
	request := newFixture().request("openid profile email offline_access roles orders:read orders:write")

	b.ReportAllocs()
	for b.Loop() {
		if preview := EvaluateToken(request); !preview.Issued {
			b.Fatal(preview.Reason)
		}
	}
}

func BenchmarkSetPassword(b *testing.B) {
	var user User
	for b.Loop() {
		if err := user.SetPassword("correct horse battery staple"); err != nil {
			b.Fatal(err)
		}
	}
}
