package model

import "time"

// SignInThrottle counts the sign-in attempts made against one key — an
// address typed, a network address, a known device — and, once too many have
// failed, makes the next wait, longer each time. Throttled instead of locked:
// a stranger's failures slow a key down but never shut its owner out of a
// device they have signed in on before.
type SignInThrottle struct {
	Base

	Key          string `gorm:"size:160;not null;uniqueIndex"`
	Failures     int    `gorm:"not null"`
	BlockedUntil *time.Time
	// ExpiresAt forgets the count a while after the last attempt.
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (SignInThrottle) TableName() string {
	return "sign_in_throttles"
}

// ThrottleKey is a key an attempt is counted against, and how many failures
// it takes before attempts wait.
type ThrottleKey struct {
	Key  string
	Free int
}

const (
	// ThrottleFree is how many failures an address typed or a device has
	// before attempts wait; a network address, shared by an office behind
	// one router, has ThrottleFreeNetwork.
	ThrottleFree        = 5
	ThrottleFreeNetwork = 50

	// The first wait, doubled at each failure past the free ones up to
	// ThrottleLongest.
	ThrottleFirst   = 30 * time.Second
	ThrottleLongest = 15 * time.Minute

	// ThrottleMemory is how long a count is kept after its last attempt.
	ThrottleMemory = 24 * time.Hour
)

// ThrottleWait is how long the next attempt waits once `failures` have been
// counted against a key that allows `free`: none until the free ones are
// used, then ThrottleFirst, doubling.
func ThrottleWait(failures, free int) time.Duration {
	over := failures - free + 1
	if over <= 0 {
		return 0
	}

	wait := ThrottleFirst
	for range over - 1 {
		wait *= 2
		if wait >= ThrottleLongest {
			return ThrottleLongest
		}
	}

	return wait
}
