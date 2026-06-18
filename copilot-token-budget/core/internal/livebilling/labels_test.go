package livebilling

import (
	"testing"
	"time"
)

func TestDisplayLabel(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	if got := DisplayLabel(nil, now); got != "(estimated)" {
		t.Fatalf("nil label = %q", got)
	}
	if got := DisplayLabel(&OrgBillingSnapshot{Availability: AvailabilityUnavailable}, now); got != "(unavailable)" {
		t.Fatalf("unavailable label = %q", got)
	}
	got := DisplayLabel(&OrgBillingSnapshot{
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC),
	}, now)
	if got != "(org aggregate, ~2h ago)" {
		t.Fatalf("available label = %q", got)
	}
}
