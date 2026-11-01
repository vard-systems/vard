package jose

import "testing"

// Every token response signs one or two JWTs, and every API call an
// application makes verifies one: these are the cost of each, per algorithm.

func BenchmarkSign(b *testing.B) {
	claims := map[string]any{"iss": "https://id.example.com", "sub": "user", "aud": "api", "exp": 2000000000, "scope": "openid orders:read"}

	for _, alg := range Algorithms {
		b.Run(alg, func(b *testing.B) {
			private, err := Generate(alg)
			if err != nil {
				b.Fatal(err)
			}
			key := Key{ID: "k1", Algorithm: alg, Private: private}

			b.ReportAllocs()
			for b.Loop() {
				if _, err := Sign(key, "at+jwt", claims); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkVerify(b *testing.B) {
	claims := map[string]any{"iss": "https://id.example.com", "sub": "user", "aud": "api", "exp": 2000000000}

	for _, alg := range Algorithms {
		b.Run(alg, func(b *testing.B) {
			private, err := Generate(alg)
			if err != nil {
				b.Fatal(err)
			}
			key := Key{ID: "k1", Algorithm: alg, Private: private}
			token, err := Sign(key, "at+jwt", claims)
			if err != nil {
				b.Fatal(err)
			}
			lookup := func(string) (Key, bool) { return key, true }

			b.ReportAllocs()
			for b.Loop() {
				var out map[string]any
				if _, err := Verify(token, lookup, &out); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
