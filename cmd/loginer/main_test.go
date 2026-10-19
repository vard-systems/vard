package main

import (
	"net/http"
	"testing"
	"time"
)

// Every connection either server takes has to have a deadline on each part of
// it. Without one a caller holds a connection, and the goroutine serving it,
// for as long as it cares to — which is why this checks the server rather
// than trusting the four fields to stay where they were put.
func TestAServerBoundsEveryPartOfAConnection(t *testing.T) {
	server := newServer(":8080", http.NewServeMux())

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{"ReadHeaderTimeout", server.ReadHeaderTimeout},
		{"ReadTimeout", server.ReadTimeout},
		{"WriteTimeout", server.WriteTimeout},
		{"IdleTimeout", server.IdleTimeout},
	}

	for _, timeout := range timeouts {
		if timeout.value <= 0 {
			t.Errorf("%s is %v, want a deadline", timeout.name, timeout.value)
		}
	}

	// Answering can mean calling an identity provider several times over, so
	// writing has to outlast reading by a good margin.
	if server.WriteTimeout <= server.ReadTimeout {
		t.Errorf("WriteTimeout %v is not longer than ReadTimeout %v", server.WriteTimeout, server.ReadTimeout)
	}
}
