package caching

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWriteRequestCheck(t *testing.T) {
	tests := []struct {
		name    string
		req     writeRequest
		wantTTL time.Duration
		wantErr bool
	}{
		{name: "an object, keeping its time", req: writeRequest{Value: json.RawMessage(`{"name":"Acme"}`)}},
		{name: "a list with a minute", req: writeRequest{Value: json.RawMessage(`["en"]`), TTLSeconds: 60}, wantTTL: time.Minute},
		{name: "no value", req: writeRequest{}, wantErr: true},
		{name: "not JSON", req: writeRequest{Value: json.RawMessage(`{name}`)}, wantErr: true},
		{name: "null", req: writeRequest{Value: json.RawMessage(` null `)}, wantErr: true},
		{name: "a negative time", req: writeRequest{Value: json.RawMessage(`1`), TTLSeconds: -1}, wantErr: true},
		{name: "more than a day", req: writeRequest{Value: json.RawMessage(`1`), TTLSeconds: 86401}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ttl, err := tt.req.check()
			if (err != nil) != tt.wantErr {
				t.Fatalf("check() error = %v, want an error: %v", err, tt.wantErr)
			}
			if ttl != tt.wantTTL {
				t.Errorf("ttl = %v, want %v", ttl, tt.wantTTL)
			}
		})
	}
}

func TestKeyNamesAndKinds(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{name: "a key", err: checkName("cache:languages:v0:all")},
		{name: "no key", err: checkName(""), wantErr: true},
		{name: "a kind", err: checkKind("session")},
		{name: "no kind", err: checkKind("")},
		{name: "an unknown kind", err: checkKind("everything"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.err != nil) != tt.wantErr {
				t.Errorf("error = %v, want an error: %v", tt.err, tt.wantErr)
			}
		})
	}
}
