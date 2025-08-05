package timezone_test

import (
	"oil/shared/timezone"
	"testing"
	"time"
)

func TestTimezoneInit(t *testing.T) {
	// Test Now() function
	now := timezone.Now()
	if now.IsZero() {
		t.Error("Now() returned zero time")
	}

	// Test GetLocation()
	loc := timezone.GetLocation()
	if loc == nil {
		t.Error("GetLocation() returned nil")
	}

	t.Logf("Current timezone: %s", loc.String())
}

func TestTimezoneWithStandardLocation(t *testing.T) {
	utcTime := time.Now().UTC()
	appTime := timezone.ToAppTime(utcTime)

	if appTime.Location() == nil {
		t.Error("Expected converted time to have a location")
	}

	t.Logf("UTC time: %s", utcTime.Format("2006-01-02 15:04:05 MST"))
	t.Logf("App time: %s", appTime.Format("2006-01-02 15:04:05 MST"))
}

func TestTimezoneFormat(t *testing.T) {
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	formatted := timezone.Format(testTime, "2006-01-02 15:04:05 MST")

	if formatted == "" {
		t.Error("Format() returned empty string")
	}

	t.Logf("Formatted time: %s", formatted)

	parsed, err := timezone.Parse("2006-01-02", "2024-01-01")
	if err != nil {
		t.Errorf("Parse() failed: %v", err)
	}

	t.Logf("Parsed time: %s", parsed.Format("2006-01-02 15:04:05 MST"))
}
