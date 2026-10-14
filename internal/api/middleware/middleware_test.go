package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestChainRunsEverythingInOrder checks that the chain is what it says: the
// logger, then recovery, then whatever was handed in — and that the one that
// was handed in is actually used, rather than quietly dropped.
func TestChainRunsEverythingInOrder(t *testing.T) {
	var order []string

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	passed := func(c *gin.Context) {
		order = append(order, "passed in")
		c.Next()
	}

	chain := Chain(log, passed)
	if len(chain) != 5 {
		t.Fatalf("Chain() has %d handlers, want 5", len(chain))
	}

	r := gin.New()
	r.Use(chain...)
	r.GET("/thing", func(c *gin.Context) {
		order = append(order, "route")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/thing", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	want := []string{"passed in", "route"}
	if len(order) != len(want) {
		t.Fatalf("ran %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("ran %v, want %v", order, want)
			break
		}
	}
}

// The handler that was passed in can stop the request, the way a preflight
// answer does, and nothing further runs.
func TestChainLetsThePassedHandlerAbort(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reached := false

	stop := func(c *gin.Context) { c.AbortWithStatus(http.StatusNoContent) }

	r := gin.New()
	r.Use(Chain(log, stop)...)
	r.GET("/thing", func(*gin.Context) { reached = true })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/thing", nil))

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if reached {
		t.Error("the route ran, want it stopped by the middleware")
	}
}

// A panic anywhere behind the chain becomes a 500 rather than a dropped
// connection, and the request is still logged.
func TestChainRecoversFromAPanic(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	nothing := func(c *gin.Context) { c.Next() }

	r := gin.New()
	r.Use(Chain(log, nothing)...)
	r.GET("/boom", func(*gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
