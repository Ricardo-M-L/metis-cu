package platform

import (
	"errors"
	"testing"
	"time"
)

// TestCachedFrontmost_MissUntilStored: a fresh process / invalidation
// must report cache miss until the first store.
func TestCachedFrontmost_MissUntilStored(t *testing.T) {
	invalidateFrontmostCache()
	if _, _, _, ok := cachedFrontmost(); ok {
		t.Fatal("expected miss on empty cache")
	}
	storeFrontmost("Safari", TierRead, nil)
	name, tier, err, ok := cachedFrontmost()
	if !ok {
		t.Fatal("expected hit immediately after store")
	}
	if name != "Safari" || tier != TierRead || err != nil {
		t.Errorf("got (%q, %q, %v), want (Safari, read, nil)", name, tier, err)
	}
}

// TestCachedFrontmost_CachesError: errors are cached too — a wedged
// osascript today is likely wedged 50ms from now, so re-shelling on
// every call wastes time. The cache delivers the same error so the
// gate keeps failing closed consistently.
func TestCachedFrontmost_CachesError(t *testing.T) {
	invalidateFrontmostCache()
	want := errors.New("osascript wedged")
	storeFrontmost("", TierFull, want)
	_, _, gotErr, ok := cachedFrontmost()
	if !ok {
		t.Fatal("expected cache hit")
	}
	if gotErr == nil || gotErr.Error() != want.Error() {
		t.Errorf("got %v, want %v", gotErr, want)
	}
}

// TestCachedFrontmost_ExpiresAfterTTL: bump the TTL down to 5ms,
// store, sleep past it, expect miss. Restores the original TTL after.
func TestCachedFrontmost_ExpiresAfterTTL(t *testing.T) {
	orig := frontmostCacheTTL
	frontmostCacheTTL = 5 * time.Millisecond
	defer func() { frontmostCacheTTL = orig }()

	invalidateFrontmostCache()
	storeFrontmost("Safari", TierRead, nil)
	time.Sleep(20 * time.Millisecond)
	if _, _, _, ok := cachedFrontmost(); ok {
		t.Fatal("expected miss after TTL expiry")
	}
}
