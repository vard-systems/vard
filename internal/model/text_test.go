package model

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// What goes through Truncate is a name a provider sent and a User-Agent
// header, and what comes out goes into a column. Postgres refuses a string
// that is not UTF-8, so a cut through the middle of a letter — or a byte the
// caller never should have sent — would fail the sign-in itself rather than
// shorten the name it was carrying.
func TestTruncateLeavesTextADatabaseWillTake(t *testing.T) {
	tests := []struct {
		name  string
		value string
		max   int
		want  string
	}{
		{name: "shorter than the limit", value: "Ada", max: 100, want: "Ada"},
		{name: "exactly the limit", value: "Ada", max: 3, want: "Ada"},
		{name: "cut where every letter is one byte", value: "Lovelace", max: 4, want: "Love"},
		{
			// Six two-byte letters: the limit falls inside the fourth.
			name:  "a cut that would land inside a letter",
			value: "Августа",
			max:   7,
			want:  "Авг",
		},
		{
			// Three-byte letters, a limit that is a multiple of three.
			name:  "a cut that lands between letters",
			value: "日本語の名前",
			max:   9,
			want:  "日本語",
		},
		{name: "a limit of nothing", value: "Ада", max: 0, want: ""},
		{
			name:  "bytes that are not UTF-8 at all",
			value: "Ada\xffLovelace",
			max:   100,
			want:  "AdaLovelace",
		},
		{
			name:  "bytes that are not UTF-8, and too long as well",
			value: "\xffАвгуста",
			max:   4,
			want:  "Ав",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.value, tt.max)

			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.value, tt.max, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Truncate(%q, %d) = %q, which is not valid UTF-8", tt.value, tt.max, got)
			}
			if len(got) > tt.max {
				t.Errorf("Truncate(%q, %d) = %q, which is %d bytes", tt.value, tt.max, got, len(got))
			}
		})
	}
}

// A name long enough to be cut is the case that used to fail: sixty Cyrillic
// letters are a hundred and twenty bytes, so the hundred-byte limit the
// providers' names are held to lands inside a letter.
func TestTruncateAName(t *testing.T) {
	got := Truncate(strings.Repeat("я", 60), 100)

	if !utf8.ValidString(got) {
		t.Fatalf("a cut name is not valid UTF-8: %q", got)
	}
	if want := strings.Repeat("я", 50); got != want {
		t.Errorf("Truncate = %q, want the whole letters that fit", got)
	}
}
