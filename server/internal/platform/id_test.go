package platform_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

var uuidV7Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// RFC 9562 Appendix A.6: Tuesday, February 22, 2022 2:22:22.00 PM GMT-05:00
// encodes as 017F22E2-79B0-7CC3-98C4-DC0C0C07398F; only the first 48 bits
// (the Unix milliseconds) are deterministic.
func TestNewIDIsUUIDv7CarryingTheClockMilliseconds(t *testing.T) {
	clock := platform.NewFakeClock(time.Date(2022, 2, 22, 19, 22, 22, 0, time.UTC))
	ids := platform.NewIDGenerator(clock)

	id := ids.New().String()

	if !uuidV7Pattern.MatchString(id) {
		t.Fatalf("ID %q is not a UUIDv7 in canonical form", id)
	}
	if got, want := id[:13], "017f22e2-79b0"; got != want {
		t.Errorf("timestamp bits = %s; want %s", got, want)
	}
}

func TestNewIDsAreUniqueWithinOneMillisecond(t *testing.T) {
	ids := platform.NewIDGenerator(platform.NewFakeClock(time.Unix(1_700_000_000, 0)))

	seen := make(map[platform.ID]bool)
	for range 10_000 {
		id := ids.New()
		if seen[id] {
			t.Fatalf("duplicate ID %s", id)
		}
		seen[id] = true
	}
}

func TestFakeClockAdvances(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	clock := platform.NewFakeClock(start)

	clock.Advance(7 * 24 * time.Hour)

	if got, want := clock.Now(), time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("Now() = %v; want %v", got, want)
	}
}

func TestParseIDRoundTripsTheCanonicalForm(t *testing.T) {
	ids := platform.NewIDGenerator(platform.SystemClock{})
	want := ids.New()

	got, err := platform.ParseID(want.String())
	if err != nil || got != want {
		t.Fatalf("ParseID(%s) = %s, %v; want %s", want, got, err, want)
	}
	if upper, err := platform.ParseID("0190B6C4-0000-7000-8000-00000000000A"); err != nil ||
		upper.String() != "0190b6c4-0000-7000-8000-00000000000a" {
		t.Errorf("ParseID(uppercase) = %s, %v; want it accepted", upper, err)
	}
}

func TestParseIDRefusesNonUUIDs(t *testing.T) {
	for _, s := range []string{"", "0190b6c4", "0190b6c4-0000-7000-8000-00000000000g",
		"0190b6c400007000800000000000000a", "{0190b6c4-0000-7000-8000-00000000000a}"} {
		if _, err := platform.ParseID(s); err == nil {
			t.Errorf("ParseID(%q) = nil error; want refused", s)
		}
	}
}
