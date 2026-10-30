package model

import (
	"strings"
	"testing"
)

func TestLanguageValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Language)
		want   string // the message, or "" when it is fine
	}{
		{name: "the language a new installation starts with", change: func(*Language) {}},
		{
			name:   "a language with a region",
			change: func(l *Language) { l.Code = "pt-BR"; l.Name = "Portuguese (Brazil)"; l.NativeName = "Português" },
		},
		{
			name:   "a language nobody has turned on",
			change: func(l *Language) { l.Code = "ky"; l.IsDefault = false; l.IsEnabled = false },
		},
		{
			name:   "no code",
			change: func(l *Language) { l.Code = " " },
			want:   "code is required",
		},
		{
			name:   "a code that is a file path",
			change: func(l *Language) { l.Code = "../en" },
			want:   "code must be a language tag, such as en, ky or pt-BR",
		},
		{
			name:   "a code in capitals",
			change: func(l *Language) { l.Code = "EN" },
			want:   "code must be a language tag, such as en, ky or pt-BR",
		},
		{
			name:   "no name",
			change: func(l *Language) { l.Name = "  " },
			want:   "name is required",
		},
		{
			name:   "no native name",
			change: func(l *Language) { l.NativeName = "" },
			want:   "native name is required",
		},
		{
			name:   "a name counted in letters, not bytes",
			change: func(l *Language) { l.NativeName = strings.Repeat("ы", 64) },
		},
		{
			name:   "a name that is too long",
			change: func(l *Language) { l.Name = strings.Repeat("a", 65) },
			want:   "name must be at most 64 characters",
		},
		{
			name:   "a position before the first",
			change: func(l *Language) { l.Position = -1 },
			want:   "position cannot be negative",
		},
		{
			name:   "turning off the language somebody sees before choosing",
			change: func(l *Language) { l.IsEnabled = false },
			want:   "the default language cannot be turned off",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			language := DefaultLanguage()
			tt.change(&language)

			err := language.Validate()

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

func TestTooLongMessage(t *testing.T) {
	tests := []struct {
		name     string
		messages map[string]string
		want     string
	}{
		{name: "nothing translated yet", messages: map[string]string{}},
		{name: "the longest a message may be", messages: map[string]string{"a": strings.Repeat("ы", MaxMessageLength)}},
		{name: "a message that is a document", messages: map[string]string{"a": strings.Repeat("a", MaxMessageLength+1)}, want: "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TooLongMessage(tt.messages); got != tt.want {
				t.Errorf("TooLongMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
