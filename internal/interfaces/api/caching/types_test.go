package caching

import (
	"testing"

	"vard/internal/interfaces/api/kit/tstypes"
)

// The panel's types for these answers are written by hand in
// web/console/src/lib/api/types.ts; this keeps the two in step.
func TestAnswersMatchThePanelsTypes(t *testing.T) {
	tstypes.Match(t, "CacheOverview", overviewResponse{})
	tstypes.Match(t, "CacheGroup", groupResponse{})
	tstypes.Match(t, "CacheDatabase", databaseResponse{})
	tstypes.Match(t, "CacheKey", keyResponse{})
	tstypes.Match(t, "CacheKeyPage", keysResponse{})
	tstypes.Match(t, "CacheKeyValue", valueResponse{})
}
