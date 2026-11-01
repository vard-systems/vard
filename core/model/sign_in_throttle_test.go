package model

import (
	"testing"
	"time"
)

func TestThrottleWaitsLongerAtEachFailure(t *testing.T) {
	cases := []struct {
		name     string
		failures int
		want     time.Duration
	}{
		{"failures still free", ThrottleFree - 1, 0},
		{"the last free one", ThrottleFree, 30 * time.Second},
		{"the first past the free ones", ThrottleFree + 1, time.Minute},
		{"the third past", ThrottleFree + 3, 4 * time.Minute},
		{"many more, held to the longest", ThrottleFree + 40, ThrottleLongest},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ThrottleWait(c.failures, ThrottleFree); got != c.want {
				t.Errorf("ThrottleWait(%d) = %v, want %v", c.failures, got, c.want)
			}
		})
	}
}
