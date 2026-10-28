package api

import (
	"regexp"
	"strconv"
	"testing"

	"vard/internal/model"
)

// The pages say how long a password must be before a form is sent; what they
// say is the server's rule, so a change to one side fails here until the
// other follows.
func TestThePagesKnowThePasswordLengths(t *testing.T) {
	cases := []struct {
		file    string
		pattern string
		want    int
	}{
		{"../../web/id/src/lib/constants.ts", `passwordMin: (\d+)`, model.MinUserPasswordLength},
		{"../../web/id/src/lib/constants.ts", `passwordMax: (\d+)`, model.MaxPasswordLength},
		{"../../web/console/src/lib/constants.ts", `MIN_ADMIN_PASSWORD = (\d+)`, model.MinAdminPasswordLength},
		{"../../web/console/src/lib/constants.ts", `MIN_USER_PASSWORD = (\d+)`, model.MinUserPasswordLength},
	}

	for _, c := range cases {
		t.Run(c.file+" "+c.pattern, func(t *testing.T) {
			found := regexp.MustCompile(c.pattern).FindStringSubmatch(read(t, c.file))
			if found == nil {
				t.Fatalf("%s says nothing matching %s", c.file, c.pattern)
			}
			if got, _ := strconv.Atoi(found[1]); got != c.want {
				t.Errorf("%s says %d, the server %d", c.file, got, c.want)
			}
		})
	}
}
