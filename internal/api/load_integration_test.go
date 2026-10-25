package api

import (
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// A load test of the endpoints applications call most, against two
// instances sharing one database and Redis. It fails on any error and on a
// latency far past what a healthy server shows, and logs the numbers:
//
//	VARD_LOAD_REQUESTS=20000 make test-integration
//
// runs it harder. The default keeps the suite quick.
func TestLiveLoad(t *testing.T) {
	f := newOAuthFixture(t)
	nodes := []*oauthFixture{f, f.on(f.s.replica())}

	clientID, secret, appID := f.register(map[string]any{"name": "Load", "type": "m2m"})
	f.authorizeAPI(appID, "orders:read")

	verifier, challenge := pkce(t)
	user := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier)
	if user.status != http.StatusOK {
		t.Fatalf("exchange = %d %v", user.status, user.body)
	}
	accessToken := user.str("access_token")

	requests := 600
	if n, err := strconv.Atoi(os.Getenv("VARD_LOAD_REQUESTS")); err == nil && n > 0 {
		requests = n
	}
	const workers = 32

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{MaxIdleConnsPerHost: workers, MaxConnsPerHost: workers},
	}

	type call struct {
		name string
		send func(node *oauthFixture) (*http.Request, error)
	}
	form := func(node *oauthFixture, path string, values url.Values) (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, node.s.root+path, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(secret))
		return req, nil
	}
	calls := []call{
		{"client credentials", func(node *oauthFixture) (*http.Request, error) {
			return form(node, "/oauth2/token", url.Values{"grant_type": {"client_credentials"}, "audience": {ordersAPI}})
		}},
		{"userinfo", func(node *oauthFixture) (*http.Request, error) {
			req, err := http.NewRequest(http.MethodGet, node.s.root+"/oauth2/userinfo", nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+accessToken)
			}
			return req, err
		}},
		{"jwks", func(node *oauthFixture) (*http.Request, error) {
			return http.NewRequest(http.MethodGet, node.s.root+"/.well-known/jwks.json", nil)
		}},
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			latencies := make([]time.Duration, requests)
			failures := make(chan string, requests)

			jobs := make(chan int)
			var wg sync.WaitGroup
			started := time.Now()
			for range workers {
				wg.Go(func() {
					for i := range jobs {
						req, err := c.send(nodes[i%2])
						if err != nil {
							failures <- err.Error()
							continue
						}
						at := time.Now()
						res, err := client.Do(req)
						latencies[i] = time.Since(at)
						if err != nil {
							failures <- err.Error()
							continue
						}
						res.Body.Close()
						if res.StatusCode != http.StatusOK {
							failures <- res.Status
						}
					}
				})
			}
			for i := range requests {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			elapsed := time.Since(started)
			close(failures)

			failed := 0
			for failure := range failures {
				if failed < 5 {
					t.Errorf("request failed: %s", failure)
				}
				failed++
			}
			if failed > 0 {
				t.Fatalf("%d of %d requests failed", failed, requests)
			}

			slices.Sort(latencies)
			at := func(p float64) time.Duration { return latencies[int(float64(len(latencies)-1)*p)] }
			t.Logf("%d requests, %d at once, two instances: %.0f/s, p50 %v, p95 %v, p99 %v",
				requests, workers, float64(requests)/elapsed.Seconds(),
				at(0.50).Round(time.Microsecond), at(0.95).Round(time.Microsecond), at(0.99).Round(time.Microsecond))

			if p99 := at(0.99); p99 > 2*time.Second {
				t.Errorf("p99 = %v, want well under 2s", p99)
			}
		})
	}
}
