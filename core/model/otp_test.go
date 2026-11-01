package model

import (
	"strings"
	"testing"
	"time"
)

func TestOTPSettingsValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*OTPSettings)
		want   string // the message, or "" when the settings are fine
	}{
		{
			name:   "the defaults",
			change: func(*OTPSettings) {},
		},
		{
			name:   "no wait between messages is allowed, if not advised",
			change: func(o *OTPSettings) { o.ResendSeconds = 0 },
		},
		{
			name:   "a code too short to be worth guessing at",
			change: func(o *OTPSettings) { o.CodeLength = 3 },
			want:   "code_length must be between 4 and 10",
		},
		{
			name:   "a code nobody would read",
			change: func(o *OTPSettings) { o.CodeLength = 11 },
			want:   "code_length must be between 4 and 10",
		},
		{
			name:   "a code that never expires",
			change: func(o *OTPSettings) { o.LifetimeMinutes = 0 },
			want:   "lifetime_minutes must be between 1 and 60",
		},
		{
			name:   "a code that outlives the day",
			change: func(o *OTPSettings) { o.LifetimeMinutes = 1440 },
			want:   "lifetime_minutes must be between 1 and 60",
		},
		{
			name:   "no guesses at all",
			change: func(o *OTPSettings) { o.MaxAttempts = 0 },
			want:   "max_attempts must be between 1 and 10",
		},
		{
			name:   "guesses without end",
			change: func(o *OTPSettings) { o.MaxAttempts = 100 },
			want:   "max_attempts must be between 1 and 10",
		},
		{
			name:   "a wait that is a wall",
			change: func(o *OTPSettings) { o.ResendSeconds = 3600 },
			want:   "resend_seconds must be between 0 and 900",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := DefaultOTPSettings()
			tt.change(&settings)

			err := settings.Validate()

			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nothing", err)
			case tt.want == "":
				return
			case err == nil:
				t.Fatalf("Validate() = nothing, want %q", tt.want)
			case err.Error() != tt.want:
				t.Errorf("Validate() = %q, want %q", err, tt.want)
			}
		})
	}
}

// A code is the length that was asked for, all digits, and stored as a hash
// rather than as itself.
func TestNewOTP(t *testing.T) {
	for _, length := range []int{MinOTPLength, 6, MaxOTPLength} {
		code, hash, err := NewOTP(length)
		if err != nil {
			t.Fatalf("NewOTP(%d) = %v", length, err)
		}

		if len(code) != length {
			t.Errorf("NewOTP(%d) gave %q, want %d digits", length, code, length)
		}
		if strings.Trim(code, "0123456789") != "" {
			t.Errorf("NewOTP(%d) gave %q, want digits only", length, code)
		}
		if hash == code {
			t.Errorf("NewOTP(%d) stored the code itself, not its hash", length)
		}
		if hash != HashSecret(code) {
			t.Errorf("NewOTP(%d) did not hash the code it gave", length)
		}
	}
}

func TestLoginCodeMatchesOTP(t *testing.T) {
	const sent = "418902"

	code := LoginCode{CodeHash: HashSecret(sent)}

	tests := []struct {
		name  string
		typed string
		want  bool
	}{
		{name: "the code that was sent", typed: sent, want: true},
		{name: "the code with a space in it, as a phone offers it", typed: "418 902", want: true},
		{name: "the code with a dash, as somebody would read it out", typed: "418-902", want: true},
		{name: "a different code", typed: "418903"},
		{name: "the code with a digit missing", typed: "41890"},
		{name: "nothing at all", typed: ""},
		{name: "only punctuation", typed: "---"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := code.MatchesOTP(tt.typed); got != tt.want {
				t.Errorf("MatchesOTP(%q) = %v, want %v", tt.typed, got, tt.want)
			}
		})
	}
}

func TestLoginCodeUsable(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	used := now.Add(-time.Minute)

	tests := []struct {
		name string
		code LoginCode
		want bool
	}{
		{
			name: "sent a moment ago, untouched",
			code: LoginCode{ExpiresAt: now.Add(10 * time.Minute)},
			want: true,
		},
		{
			name: "guessed at, with guesses left",
			code: LoginCode{ExpiresAt: now.Add(10 * time.Minute), Attempts: 4},
			want: true,
		},
		{
			name: "expired",
			code: LoginCode{ExpiresAt: now.Add(-time.Second)},
		},
		{
			name: "already used",
			code: LoginCode{ExpiresAt: now.Add(10 * time.Minute), UsedAt: &used},
		},
		{
			name: "guessed at too often",
			code: LoginCode{ExpiresAt: now.Add(10 * time.Minute), Attempts: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.Usable(now, 5); got != tt.want {
				t.Errorf("Usable() = %v, want %v", got, tt.want)
			}
		})
	}
}
