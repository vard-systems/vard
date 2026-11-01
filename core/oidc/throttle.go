package oidc

import (
	"context"
	"encoding/hex"
	"net"

	"vard/core/model"
)

// Users' sign-ins are throttled, not locked. Each attempt is counted, before
// it is checked, against the address typed and the caller's network; past a
// few failures each waits longer. Keyed by the address typed rather than the
// account, an address nobody has is throttled exactly like one somebody has.

const throttlePurpose = "sign-in throttle"

// signInKeys are what an attempt for `email` from `client` is counted
// against: the address typed, and the caller's network.
type signInKeys struct {
	keys []model.ThrottleKey
	// account is the key a success forgets; network is the one it only
	// gives its attempt back to.
	account string
	network model.ThrottleKey
}

func (s *Service) keysFor(email string, client Client) signInKeys {
	account := "email:" + s.accountKey(email)
	network := model.ThrottleKey{Key: "network:" + networkOf(client.IP), Free: model.ThrottleFreeNetwork}

	return signInKeys{
		keys:    []model.ThrottleKey{{Key: account, Free: model.ThrottleFree}, network},
		account: account,
		network: network,
	}
}

// reserve counts an attempt before it is checked; false when it has to wait.
func (s *Service) reserve(ctx context.Context, keys signInKeys) (bool, error) {
	return s.store.ReserveSignIn(ctx, keys.keys, s.now())
}

// succeeded forgets the attempts against the address typed, and gives the
// network back the one this took.
func (s *Service) succeeded(ctx context.Context, keys signInKeys) error {
	if err := s.store.ClearSignIn(ctx, keys.account); err != nil {
		return err
	}
	return s.store.ReturnSignIn(ctx, keys.network)
}

// passwordRight gives the network back the attempt a right password took
// when a second factor is still to come. The address typed keeps its count,
// which only the right second factor clears; the second factor's own
// attempt is counted, and given back, by itself.
func (s *Service) passwordRight(ctx context.Context, keys signInKeys) error {
	return s.store.ReturnSignIn(ctx, keys.network)
}

// accountKey names an address typed without naming it: a keyed hash of it
// as it is compared.
func (s *Service) accountKey(email string) string {
	tag := s.sealer.Tag(throttlePurpose, []byte(model.NormalizeEmail(email)))
	return hex.EncodeToString(tag[:16])
}

// networkOf groups IPv6 callers by /64, which one client can rotate through.
func networkOf(address string) string {
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() != nil {
		return address
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}
