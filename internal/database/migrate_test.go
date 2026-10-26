package database

import (
	"io"
	"os"
	"testing"
)

// goose writes a status row by row without a newline, trusting the log
// package to end each one; printed as given, the table was one line.
func TestTheStatusPrintsOneLinePerMigration(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{"a row without a newline", "    %-24s -- %v", []any{"Pending", "00002_x.go"}, "    Pending                  -- 00002_x.go\n"},
		{"a line that ends in one already", "done\n", nil, "done\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStdout(t, func() { printLogger{}.Printf(tt.format, tt.args...) })

			if got != tt.want {
				t.Errorf("printed %q, want %q", got, tt.want)
			}
		})
	}
}

func captureStdout(t *testing.T, write func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = stdout }()

	write()
	_ = writer.Close()

	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
