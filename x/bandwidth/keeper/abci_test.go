package keeper

import (
	"testing"
	"time"
)

func TestDayIndexBoundary(t *testing.T) {
	mustUnix := func(y int, m time.Month, d, h, mi, s int) int64 {
		return time.Date(y, m, d, h, mi, s, 0, time.UTC).Unix()
	}

	// 09:48 and 10:16 UTC on the same calendar date -- both fall inside the
	// same [03:00 UTC, next 03:00 UTC) window, so they must map to the same
	// day bucket (no premature re-topup).
	sameDayA := mustUnix(2026, 9, 3, 9, 48, 24)
	sameDayB := mustUnix(2026, 9, 3, 10, 16, 0)
	if dayIndex(sameDayA) != dayIndex(sameDayB) {
		t.Fatalf("expected same day bucket, got %d vs %d", dayIndex(sameDayA), dayIndex(sameDayB))
	}

	// Either side of the 03:00 UTC boundary must fall into different day
	// buckets, even though only 2 seconds apart.
	justBefore := mustUnix(2026, 9, 4, 2, 59, 59)
	justAfter := mustUnix(2026, 9, 4, 3, 0, 1)
	if dayIndex(justBefore) == dayIndex(justAfter) {
		t.Fatalf("expected different day buckets across the 03:00 UTC boundary, got same index %d", dayIndex(justBefore))
	}
	if dayIndex(justAfter) != dayIndex(sameDayA)+1 {
		t.Fatalf("expected next-day index to be exactly +1, got %d vs %d", dayIndex(justAfter), dayIndex(sameDayA))
	}
}
