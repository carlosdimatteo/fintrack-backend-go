package api

import (
	"testing"
	"time"
)

func TestSnapshotPeriod(t *testing.T) {
	cases := []struct {
		at                  string
		wantYear, wantMonth int
	}{
		{"2026-03-02T20:13:00Z", 2026, 2},  // early March closes February
		{"2026-03-10T23:59:00Z", 2026, 2},  // last day of the window
		{"2026-03-11T00:00:00Z", 2026, 3},  // first day outside it
		{"2026-06-30T22:59:00Z", 2026, 6},  // month-end accounting stays in its month
		{"2027-01-04T12:00:00Z", 2026, 12}, // January closes December of the previous year
	}
	for _, c := range cases {
		at, err := time.Parse(time.RFC3339, c.at)
		if err != nil {
			t.Fatal(err)
		}
		year, month := snapshotPeriod(at)
		if year != c.wantYear || month != c.wantMonth {
			t.Errorf("snapshotPeriod(%s) = %d/%d, want %d/%d", c.at, year, month, c.wantYear, c.wantMonth)
		}
	}
}
