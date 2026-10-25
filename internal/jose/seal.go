package jose

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
)

type Sealer struct {
	aead cipher.AEAD
	key  [sha256.Size]byte
}

func NewSealer(secret string) (*Sealer, error) {
	sum := sha256.Sum256([]byte(secret))

	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Sealer{aead: aead, key: sum}, nil
}

// Tag is a keyed hash for values that are compared, not read back, and too
// short for a bare hash to survive a database dump. `purpose` separates uses,
// and the sealing key itself is never used.
func (s *Sealer) Tag(purpose string, data []byte) []byte {
	key := sha256.Sum256(append(append([]byte{}, s.key[:]...), []byte(purpose)...))
	mac := hmac.New(sha256.New, key[:])
	mac.Write(data)

	return mac.Sum(nil)
}

func (s *Sealer) Seal(key crypto.Signer) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("jose: encode private key: %w", err)
	}

	return s.SealBytes(der)
}

func (s *Sealer) SealBytes(plain []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return s.aead.Seal(nonce, nonce, plain, nil), nil
}

func (s *Sealer) OpenBytes(sealed []byte) ([]byte, error) {
	size := s.aead.NonceSize()
	if len(sealed) < size {
		return nil, ErrWrongSecret
	}

	plain, err := s.aead.Open(nil, sealed[:size], sealed[size:], nil)
	if err != nil {
		return nil, ErrWrongSecret
	}

	return plain, nil
}

var ErrWrongSecret = errors.New("jose: a signing key could not be decrypted; is VARD_SECRET_KEY the one it was made with?")

func (s *Sealer) Open(sealed []byte) (crypto.Signer, error) {
	der, err := s.OpenBytes(sealed)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("jose: decode private key: %w", err)
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("jose: stored key of type %T cannot sign", key)
	}

	return signer, nil
}
