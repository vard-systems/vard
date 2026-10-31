package apis

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
)

func TestAPIRequestApplyTo(t *testing.T) {
	existing := uuid.New()

	stored := func() *model.API {
		api := &model.API{
			Identifier:       "https://api.shop.com/orders",
			SigningAlgorithm: model.AlgRS256,
			Scopes:           []model.APIScope{{Name: "orders:read"}},
		}
		api.Scopes[0].ID = existing
		return api
	}

	tests := []struct {
		name     string
		creating bool
		request  apiRequest
		want     string
	}{
		{
			name:     "a new API, tidied",
			creating: true,
			request: apiRequest{
				Name:       " Orders ",
				Identifier: " https://api.shop.com/orders ",
				Scopes:     []scopeRequest{{Name: " Orders:Read "}},
			},
		},
		{
			name:     "a new API without an identifier",
			creating: true,
			request:  apiRequest{Name: "Orders"},
			want:     "identifier is required",
		},
		{
			name:    "an update keeping a scope by id and adding one",
			request: apiRequest{Name: "Orders", Scopes: []scopeRequest{{ID: existing, Name: "orders:view"}, {Name: "orders:write"}}},
		},
		{
			name:    "an update naming another API's scope",
			request: apiRequest{Name: "Orders", Scopes: []scopeRequest{{ID: uuid.New(), Name: "orders:read"}}},
			want:    "orders:read is not a scope of this API.",
		},
		{
			name:    "a scope the model refuses",
			request: apiRequest{Name: "Orders", Scopes: []scopeRequest{{Name: "openid"}}},
			want:    `scopes: "openid" is an OpenID Connect scope and cannot be an API scope`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &model.API{}
			if !tt.creating {
				api = stored()
			}

			err := tt.request.applyTo(api, tt.creating)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("applyTo() = %v, want nothing", err)
				}
				if api.Identifier != "https://api.shop.com/orders" || api.Scopes[0].Name[0] != 'o' {
					t.Errorf("api = %+v, want it tidied with the identifier kept", api)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) || fault.Message != tt.want {
				t.Fatalf("applyTo() = %v, want %q", err, tt.want)
			}
		})
	}
}
