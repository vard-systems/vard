package account

import (
	"strings"
	"testing"
)

func TestRegisterRequestRules(t *testing.T) {
	good := func() registerRequest {
		return registerRequest{Request: "handle", Email: "  Ada@Example.com ", Password: "long enough now"}
	}

	req := good()
	if err := req.validate(); err != nil {
		t.Fatalf("a good registration was refused: %v", err)
	}
	if req.Email != "ada@example.com" {
		t.Errorf("Email = %q, want it trimmed and lower cased", req.Email)
	}

	cases := map[string]func(*registerRequest){
		"no request":          func(r *registerRequest) { r.Request = "" },
		"not an email":        func(r *registerRequest) { r.Email = "ada" },
		"short password":      func(r *registerRequest) { r.Password = "short" },
		"password too long":   func(r *registerRequest) { r.Password = strings.Repeat("é", 129) },
		"first name too long": func(r *registerRequest) { r.FirstName = strings.Repeat("a", 101) },
	}

	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			req := good()
			change(&req)
			if err := req.validate(); err == nil {
				t.Error("accepted")
			}
		})
	}
}

func TestResetRequestNeedsAToken(t *testing.T) {
	req := resetRequest{Password: "long enough now"}
	if err := req.validate(); err == nil {
		t.Error("a reset without a token was accepted")
	}
}
