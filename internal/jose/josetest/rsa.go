// Package josetest makes keys for tests that would be slow to generate the
// usual way.
package josetest

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"testing"
)

// RSAKeyOfBits is an RSA key whose modulus is exactly that many bits, made
// of 512-bit primes, which takes about a second where two primes of half the
// size can take a minute. A verifier sees only n and e, so it is as good as
// any key that long. bits must be at least 2048.
func RSAKeyOfBits(t testing.TB, bits int) *rsa.PrivateKey {
	t.Helper()

	one := big.NewInt(1)
	for {
		n, primes := big.NewInt(1), []*big.Int{}
		for n.BitLen() < bits-1024 {
			p, err := rand.Prime(rand.Reader, 512)
			if err != nil {
				t.Fatal(err)
			}
			primes, n = append(primes, p), n.Mul(n, p)
		}
		last, err := rand.Prime(rand.Reader, bits-n.BitLen()+1)
		if err != nil {
			t.Fatal(err)
		}
		primes, n = append(primes, last), n.Mul(n, last)
		if n.BitLen() != bits {
			continue
		}

		phi := big.NewInt(1)
		for _, p := range primes {
			phi.Mul(phi, new(big.Int).Sub(p, one))
		}
		d := new(big.Int).ModInverse(big.NewInt(65537), phi)
		if d == nil {
			continue
		}

		private := &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n, E: 65537}, D: d, Primes: primes}
		private.Precompute()
		return private
	}
}
