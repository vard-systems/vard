package oidc

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net"
	"strings"

	"vard/internal/brand"
	"vard/internal/model"
)

// Users' sign-ins are throttled, not locked. Each attempt is counted, before
// it is checked, against the address typed and the caller's network; past a
// few failures each waits longer. Keyed by the address typed rather than the
// account, an address nobody has is throttled exactly like one somebody has.
// A browser that has signed in to the account before carries a known-device
// cookie, and is counted against itself instead of the address typed, so a
// stranger's failures elsewhere never shut the owner out of it.

const (
	throttlePurpose = "sign-in throttle"
	devicePurpose   = "known device"

	// KnownDeviceCookie is the browser's record of the accounts it has
	// signed in to.
	KnownDeviceCookie = brand.KnownDeviceCookie

	// maxKnownAccounts is how many accounts one browser remembers.
	maxKnownAccounts = 5
)

// ErrChallengeRequired is a sign-in the installation's challenge hook asks
// more of, such as a CAPTCHA, before it is checked.
var ErrChallengeRequired = problem("challenge_required")

// ChallengeHook lets an installation ask more of a sign-in once its keys have
// failed ChallengeAfter times: a CAPTCHA, a proof of work. Check answers nil
// to let it go ahead, or ErrChallengeRequired.
type ChallengeHook interface {
	Check(ctx context.Context, attempt ChallengeAttempt) error
}

// ChallengeAttempt is what a challenge hook is shown of a sign-in.
type ChallengeAttempt struct {
	IP       string
	Failures int
	// Answer is what the sign-in form sent back for the challenge, "" for
	// none.
	Answer string
}

// ChallengeAfter is how many failures go by before the hook is asked.
const ChallengeAfter = 3

type noChallenge struct{}

func (noChallenge) Check(context.Context, ChallengeAttempt) error { return nil }

// UseChallenge sets the hook sign-ins are checked by past ChallengeAfter
// failures; none asks nothing.
func (s *Service) UseChallenge(hook ChallengeHook) {
	if hook == nil {
		hook = noChallenge{}
	}
	s.challenge = hook
}

// signInKeys are what an attempt for `email` from `client` is counted
// against: the caller's network, and the address typed — or, from a browser
// that has signed in to it before, that browser.
type signInKeys struct {
	keys []model.ThrottleKey
	// cleared are the keys a success forgets; network is the one it only
	// gives its attempt back to.
	cleared []string
	network string
}

func (s *Service) keysFor(email string, client Client) signInKeys {
	account := s.accountKey(email)
	network := "network:" + networkOf(client.IP)

	own := "email:" + account
	if device, ok := s.knownDevice(client.Device, account); ok {
		own = "device:" + device + ":" + account
	}

	return signInKeys{
		keys: []model.ThrottleKey{
			{Key: own, Free: model.ThrottleFree},
			{Key: network, Free: model.ThrottleFreeNetwork},
		},
		cleared: []string{own},
		network: network,
	}
}

// reserve counts an attempt before it is checked, and asks the challenge
// hook when the keys have failed often. false when it has to wait.
func (s *Service) reserve(ctx context.Context, keys signInKeys, client Client) (bool, error) {
	attempt, err := s.store.ReserveSignIn(ctx, keys.keys, s.now())
	if err != nil || !attempt.Allowed {
		return false, err
	}

	if attempt.Failures >= ChallengeAfter {
		if err := s.challenge.Check(ctx, ChallengeAttempt{IP: client.IP, Failures: attempt.Failures, Answer: client.Challenge}); err != nil {
			return false, err
		}
	}

	return true, nil
}

// succeeded forgets the attempts against the address typed or the device, and
// gives the network back the one this took.
func (s *Service) succeeded(ctx context.Context, keys signInKeys) error {
	if err := s.store.ClearSignIn(ctx, keys.cleared...); err != nil {
		return err
	}
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

// knownDevice is the id this browser goes by for the account whose key is
// given, when its cookie says it has signed in to it before.
func (s *Service) knownDevice(cookie, account string) (string, bool) {
	for entry := range strings.SplitSeq(cookie, "~") {
		key, rest, ok := strings.Cut(entry, ".")
		if !ok || key != account {
			continue
		}
		device, mac, ok := strings.Cut(rest, ".")
		if !ok {
			continue
		}
		// Signed with the current key, or one rotated out since.
		for _, tag := range s.sealer.Tags(devicePurpose, []byte(account+"."+device)) {
			if hmac.Equal([]byte(mac), []byte(base64.RawURLEncoding.EncodeToString(tag[:16]))) {
				return device, true
			}
		}
	}
	return "", false
}

// rememberDevice is the cookie that also names this browser for the account
// just signed in to, newest first, keeping maxKnownAccounts.
func (s *Service) rememberDevice(cookie, email string) (string, error) {
	account := s.accountKey(email)

	device, known := s.knownDevice(cookie, account)
	if !known {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return "", err
		}
		device = base64.RawURLEncoding.EncodeToString(raw)
	}

	entries := []string{account + "." + device + "." + s.deviceMAC(account, device)}
	for entry := range strings.SplitSeq(cookie, "~") {
		key, _, _ := strings.Cut(entry, ".")
		if entry != "" && key != account && len(entries) < maxKnownAccounts {
			if _, ok := s.knownDevice(entry, key); ok {
				entries = append(entries, entry)
			}
		}
	}

	return strings.Join(entries, "~"), nil
}

func (s *Service) deviceMAC(account, device string) string {
	tag := s.sealer.Tag(devicePurpose, []byte(account+"."+device))
	return base64.RawURLEncoding.EncodeToString(tag[:16])
}
