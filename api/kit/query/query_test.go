package query

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func contextFor(rawQuery string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)

	return c
}

func TestInt(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		{"", 50},
		{"limit=20", 20},
		{"limit=900", 200},
		{"limit=-1", 50},
		{"limit=ten", 50},
	}

	for _, tt := range tests {
		if got := Int(contextFor(tt.query), "limit", 50, 200); got != tt.want {
			t.Errorf("Int(%q) = %d, want %d", tt.query, got, tt.want)
		}
	}
}

func TestBool(t *testing.T) {
	if got := Bool(contextFor("enabled=true"), "enabled"); got == nil || !*got {
		t.Errorf("Bool(true) = %v, want true", got)
	}
	if got := Bool(contextFor("enabled=false"), "enabled"); got == nil || *got {
		t.Errorf("Bool(false) = %v, want false", got)
	}
	if got := Bool(contextFor("enabled=yes"), "enabled"); got != nil {
		t.Errorf("Bool(yes) = %v, want nil", *got)
	}
}
