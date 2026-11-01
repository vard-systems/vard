package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"vard/internal/brand"
)

// The scripts (make dev, make test-integration, the db targets) read .env with
// load_env in scripts/lib.sh and export it before starting the server, which
// then believes the environment over the file. If the two readings ever
// differ, the server started by a script runs with a different value from the
// one started by `make run` — for VARD_SECRET_KEY, a different key, and
// signing keys one of them cannot open. So every way of writing a line is read
// both ways here and has to come out the same.
func TestTheScriptsReadDotEnvLikeTheServer(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}

	lib, err := filepath.Abs(filepath.Join("..", "..", "scripts", "lib.sh"))
	if err != nil {
		t.Fatal(err)
	}

	name := env("SECRET_KEY")

	tests := []struct {
		name string
		env  string
	}{
		{"a base64 key, ending in one =", name + "=abc/def+ghi0123456789abcdefghijklmnopqrstu=\n"},
		{"a value ending in ==", name + "=abc0123456789abcdefghijklmnopqrstuvwx==\n"},
		{"an = inside the value", name + "=a=b=c\n"},
		{"double quotes", name + `="quoted value"` + "\n"},
		{"single quotes", name + "='quoted value'\n"},
		{"export in front", "export " + name + "=exported\n"},
		{"spaces around the =", name + " = spaced\n"},
		{"a comment after the value", name + "=commented # not part of it\n"},
		{"Windows line endings", name + "=crlf=\r\n"},
		{"the same name twice", name + "=first=\n" + name + "=second=\n"},
		{"an empty value", name + "=\n"},
		{"no final newline", name + "=unterminated="},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writeEnv(t, test.env)
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			v, err := read()
			if err != nil {
				t.Fatal(err)
			}
			server := v.GetString(name)

			script := exec.Command(bash, "-c", `. "$1" >/dev/null && cd "$2" && load_env && printf '%s' "${`+name+`-}"`, "bash", lib, dir)
			script.Env = withoutPrefixedVariables(os.Environ())
			out, err := script.Output()
			if err != nil {
				t.Fatalf("load_env: %v", err)
			}

			if got := string(out); got != server {
				t.Errorf("load_env read %q, the server %q", got, server)
			}
		})
	}
}

func withoutPrefixedVariables(environ []string) []string {
	var kept []string
	for _, pair := range environ {
		if !strings.HasPrefix(pair, brand.EnvPrefix) {
			kept = append(kept, pair)
		}
	}
	return kept
}
