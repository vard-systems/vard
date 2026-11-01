package store

import (
	"reflect"
	"testing"

	"vard/core/model"
)

// Whatever ends at expires_at is deleted once it has: a model given an
// ExpiresAt and left out of the sweep would keep every row it ever had.
func TestEverythingThatExpiresIsSwept(t *testing.T) {
	swept := map[reflect.Type]bool{}
	for _, table := range expiring {
		swept[reflect.TypeOf(table)] = true
	}

	for _, m := range model.All() {
		kind := reflect.TypeOf(m)
		if _, ok := kind.Elem().FieldByName("ExpiresAt"); ok && !swept[kind] {
			t.Errorf("%s has an ExpiresAt but is not in expiring", kind.Elem().Name())
		}
	}
}
