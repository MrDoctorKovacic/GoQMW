package gps

import (
	"testing"
	"time"
)

func TestProcessTimezone(t *testing.T) {

	laTime, _ := time.LoadLocation("America/Los_Angeles")
	ceTime, _ := time.LoadLocation("America/Denver")
	eaTime, _ := time.LoadLocation("America/New_York")

	tables := []struct {
		input  *Loc
		output *time.Location
	}{
		{&Loc{CurrentFix: Fix{Latitude: "34.0522", Longitude: "-118.2437"}}, laTime},
		{&Loc{CurrentFix: Fix{Latitude: "39.7392", Longitude: "-104.9903"}}, ceTime},
		{&Loc{CurrentFix: Fix{Latitude: "25.7617", Longitude: "-80.1918"}}, eaTime},
	}

	for _, table := range tables {
		Location.CurrentFix.Latitude = table.input.CurrentFix.Latitude
		Location.CurrentFix.Longitude = table.input.CurrentFix.Longitude
		processTimezone()
		if Location.Timezone.String() != table.output.String() {
			t.Errorf("processTimezone() = %s; want %s", table.input.Timezone.String(), table.output.String())
		}
	}
}
