package jose

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
)

// A Sealer keeps secrets at rest. Each value is sealed with AES-256-GCM under
// a key derived from VARD_SECRET_KEY with HKDF, and starts with a version and
// the key's id, so a key can be rotated: the current one seals, a previous
// one only opens. Each value is bound to where it is kept — its table, column
// and row — as additional data, so one moved anywhere else does not open.
//
// While VARD_SECRET_KEY_PREVIOUS is set, the server seals every stored value
// again under the current key as it starts; `secrets reseal` does the same
// offline.

// sealedVersion starts every sealed value: the format, then the key's id.
var sealedVersion = []byte{'v', 1}

const keyIDSize = 4

// Context is where a sealed value is kept, bound into it: Bind's answer.
type Context string

// Bind is the context of a value kept in table.column of the row with the
// given id (or other key, for a row known by one).
func Bind(table, column string, row any) Context {
	return Context(fmt.Sprintf("%s.%s:%v", table, column, row))
}

type sealingKey struct {
	id   []byte
	aead cipher.AEAD
	// tag keys the hashes of values compared rather than read back.
	tag []byte
}

type Sealer struct {
	current  sealingKey
	previous []sealingKey
	// tagKey keys Tag once UseTagKey has given it the installation's own;
	// until then the current key's is used.
	tagKey []byte
}

// NewSealer seals with secret and opens with it or any of previous, the
// secrets it replaced.
func NewSealer(secret string, previous ...string) (*Sealer, error) {
	current, err := deriveKey(secret)
	if err != nil {
		return nil, err
	}

	s := &Sealer{current: current}
	for _, old := range previous {
		if old == "" || old == secret {
			continue
		}
		key, err := deriveKey(old)
		if err != nil {
			return nil, err
		}
		s.previous = append(s.previous, key)
	}

	return s, nil
}

func deriveKey(secret string) (sealingKey, error) {
	derive := func(label string, size int) ([]byte, error) {
		return hkdf.Key(sha256.New, []byte(secret), nil, "sealer/"+label, size)
	}

	sealing, err := derive("seal", 32)
	if err != nil {
		return sealingKey{}, err
	}
	id, err := derive("key id", keyIDSize)
	if err != nil {
		return sealingKey{}, err
	}
	tag, err := derive("tag", 32)
	if err != nil {
		return sealingKey{}, err
	}

	aead, err := newGCM(sealing)
	if err != nil {
		return sealingKey{}, err
	}

	return sealingKey{id: id, aead: aead, tag: tag}, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Tag is a keyed hash for values that are compared, not read back, and too
// short for a bare hash to survive a database dump. `purpose` separates uses.
// It is keyed by the installation's own tag key (UseTagKey), which is kept
// sealed and resealed with everything else, so rotating VARD_SECRET_KEY
// leaves every stored tag — a recovery code's — as it was.
func (s *Sealer) Tag(purpose string, data []byte) []byte {
	key := s.tagKey
	if key == nil {
		key = s.current.tag
	}
	return keyedHash(key, purpose, data)
}

// UseTagKey makes Tag keyed by the installation's own tag key from now on.
func (s *Sealer) UseTagKey(key []byte) {
	s.tagKey = append([]byte{}, key...)
}

func keyedHash(key []byte, purpose string, data []byte) []byte {
	return keyedHashRaw(append(append([]byte{}, key...), []byte(purpose)...), data)
}

func keyedHashRaw(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// SealFor seals plain, bound to where it will be kept.
func (s *Sealer) SealFor(where Context, plain []byte) ([]byte, error) {
	nonce := make([]byte, s.current.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	out := append(append(append([]byte{}, sealedVersion...), s.current.id...), nonce...)
	return s.current.aead.Seal(out, nonce, plain, []byte(where)), nil
}

// OpenFor opens a value kept where it says. stale is true when it was sealed
// under a previous key, and is worth sealing again.
func (s *Sealer) OpenFor(where Context, sealed []byte) (plain []byte, stale bool, err error) {
	if body, ok := bytes.CutPrefix(sealed, sealedVersion); ok && len(body) > keyIDSize {
		id, rest := body[:keyIDSize], body[keyIDSize:]
		for i, key := range append([]sealingKey{s.current}, s.previous...) {
			if !bytes.Equal(id, key.id) {
				continue
			}
			if plain, err := openWith(key.aead, rest, []byte(where)); err == nil {
				return plain, i > 0, nil
			}
		}
	}

	return nil, false, ErrWrongSecret
}

func openWith(aead cipher.AEAD, sealed, context []byte) ([]byte, error) {
	size := aead.NonceSize()
	if len(sealed) < size {
		return nil, ErrWrongSecret
	}
	return aead.Open(nil, sealed[:size], sealed[size:], context)
}

var ErrWrongSecret = errors.New("jose: a sealed value could not be opened; is VARD_SECRET_KEY (or VARD_SECRET_KEY_PREVIOUS) the one it was sealed with, and is it where it was sealed for?")

// Seal seals a private key, bound to where it will be kept.
func (s *Sealer) Seal(where Context, key crypto.Signer) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("jose: encode private key: %w", err)
	}

	return s.SealFor(where, der)
}

// Open opens a private key kept where it says, and says whether it is stale.
func (s *Sealer) Open(where Context, sealed []byte) (crypto.Signer, bool, error) {
	der, stale, err := s.OpenFor(where, sealed)
	if err != nil {
		return nil, false, err
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, false, fmt.Errorf("jose: decode private key: %w", err)
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, false, fmt.Errorf("jose: stored key of type %T cannot sign", key)
	}

	return signer, stale, nil
}
