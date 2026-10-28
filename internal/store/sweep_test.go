package store

import (
	"reflect"
	"slices"
	"testing"

	"vard/internal/model"
)

// Every table whose rows end at expires_at is swept, or it grows for ever
// with rows nothing reads again.
func TestEveryExpiringModelIsSwept(t *testing.T) {
	swept := []reflect.Type{}
	for _, table := range expiring {
		swept = append(swept, reflect.TypeOf(table).Elem())
	}

	for _, table := range model.All() {
		kind := reflect.TypeOf(table).Elem()
		if _, ok := kind.FieldByName("ExpiresAt"); ok && !slices.Contains(swept, kind) {
			t.Errorf("model.%s has an ExpiresAt but is not in the sweep's list", kind.Name())
		}
	}
}
