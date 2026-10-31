package migrations

import "testing"

// goose applies migrations by version, so a gap or a repeat would leave one
// out or run two in an order nobody chose.
func TestTheVersionsCountUpFromTheSetup(t *testing.T) {
	for i, migration := range All() {
		if want := int64(i + 1); migration.Version != want {
			t.Errorf("migration %d is version %d, want %d", i, migration.Version, want)
		}
	}
}
