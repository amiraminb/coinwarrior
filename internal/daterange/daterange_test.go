package daterange

import (
	"testing"
	"time"
)

func mustDay(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil {
		t.Fatalf("mustDay(%q): %v", value, err)
	}
	return parsed
}

// Wednesday, with a time component to confirm Resolve normalizes to whole days.
var resolveNow = time.Date(2026, 3, 18, 14, 30, 0, 0, time.UTC)

// A January reference date for exercising year-boundary rollover.
var janNow = time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

func TestResolveKeywords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		now   time.Time
		start string
		end   string
	}{
		{"today", "today", resolveNow, "2026-03-18", "2026-03-18"},
		{"yesterday", "yesterday", resolveNow, "2026-03-17", "2026-03-17"},
		// Week starts Monday; 2026-03-18 is a Wednesday.
		{"week", "week", resolveNow, "2026-03-16", "2026-03-22"},
		{"lastweek", "lastweek", resolveNow, "2026-03-09", "2026-03-15"},
		{"month", "month", resolveNow, "2026-03-01", "2026-03-31"},
		{"lastmonth", "lastmonth", resolveNow, "2026-02-01", "2026-02-28"},
		{"year", "year", resolveNow, "2026-01-01", "2026-12-31"},
		{"lastyear", "lastyear", resolveNow, "2025-01-01", "2025-12-31"},
		{"explicit range", "2026-04-01..2026-04-30", resolveNow, "2026-04-01", "2026-04-30"},
		{"uppercase keyword", "TODAY", resolveNow, "2026-03-18", "2026-03-18"},
		// January now() exercises the year rollover in lastmonth / lastyear.
		{"lastmonth crosses year", "lastmonth", janNow, "2025-12-01", "2025-12-31"},
		{"lastyear from january", "lastyear", janNow, "2025-01-01", "2025-12-31"},
		{"month in january", "month", janNow, "2026-01-01", "2026-01-31"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := Resolve(tc.input, tc.now)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tc.input, err)
			}
			if !start.Equal(mustDay(t, tc.start)) {
				t.Errorf("Resolve(%q) start = %s, want %s", tc.input, start.Format("2006-01-02"), tc.start)
			}
			if !end.Equal(mustDay(t, tc.end)) {
				t.Errorf("Resolve(%q) end = %s, want %s", tc.input, end.Format("2006-01-02"), tc.end)
			}
		})
	}
}

func TestNamesAreAllResolvable(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("Names() is empty")
	}
	for _, name := range names {
		if _, _, err := Resolve(name, resolveNow); err != nil {
			t.Errorf("Names() advertises %q but Resolve rejects it: %v", name, err)
		}
	}

	names[0] = "mutated"
	if Names()[0] == "mutated" {
		t.Error("Names() exposes the package-level slice; callers can mutate it")
	}
}

func TestResolveErrors(t *testing.T) {
	inputs := []string{
		"",
		"   ",
		"notarange",
		"2026-04-01..2026-04-02..2026-04-03",
		"bad..2026-04-30",
		"2026-04-01..bad",
		"2026-04-30..2026-04-01", // end before start
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, _, err := Resolve(input, resolveNow); err == nil {
				t.Errorf("Resolve(%q) = nil error, want error", input)
			}
		})
	}
}

func TestMonthBounds(t *testing.T) {
	tests := []struct {
		name  string
		month time.Time
		start string
		end   string
	}{
		{"31-day month", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), "2026-03-01", "2026-03-31"},
		{"february non-leap", time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), "2026-02-01", "2026-02-28"},
		{"february leap year", time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), "2024-02-01", "2024-02-29"},
		{"december", time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), "2026-12-01", "2026-12-31"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := MonthBounds(tc.month)
			if !start.Equal(mustDay(t, tc.start)) {
				t.Errorf("MonthBounds start = %s, want %s", start.Format("2006-01-02"), tc.start)
			}
			if !end.Equal(mustDay(t, tc.end)) {
				t.Errorf("MonthBounds end = %s, want %s", end.Format("2006-01-02"), tc.end)
			}
		})
	}
}

func TestParseMonth(t *testing.T) {
	now := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)

	got, err := ParseMonth("2026-07", now)
	if err != nil {
		t.Fatalf("ParseMonth: %v", err)
	}
	if !got.Equal(mustDay(t, "2026-07-01")) {
		t.Errorf("ParseMonth(\"2026-07\") = %s, want 2026-07-01", got.Format("2006-01-02"))
	}

	got, err = ParseMonth("", now)
	if err != nil {
		t.Fatalf("ParseMonth(empty): %v", err)
	}
	if !got.Equal(mustDay(t, "2026-03-01")) {
		t.Errorf("ParseMonth(\"\") = %s, want 2026-03-01 (current month)", got.Format("2006-01-02"))
	}

	if _, err := ParseMonth("2026-13", now); err == nil {
		t.Error("ParseMonth(\"2026-13\") = nil error, want error")
	}
	if _, err := ParseMonth("not-a-month", now); err == nil {
		t.Error("ParseMonth(\"not-a-month\") = nil error, want error")
	}
}

func TestContains(t *testing.T) {
	start := mustDay(t, "2026-03-01")
	end := mustDay(t, "2026-03-31")

	tests := []struct {
		date string
		want bool
	}{
		{"2026-03-01", true},  // inclusive lower bound
		{"2026-03-31", true},  // inclusive upper bound
		{"2026-03-15", true},  // interior
		{"2026-02-28", false}, // before
		{"2026-04-01", false}, // after
	}
	for _, tc := range tests {
		t.Run(tc.date, func(t *testing.T) {
			got, err := Contains(tc.date, start, end)
			if err != nil {
				t.Fatalf("Contains(%q): %v", tc.date, err)
			}
			if got != tc.want {
				t.Errorf("Contains(%q) = %t, want %t", tc.date, got, tc.want)
			}
		})
	}

	if _, err := Contains("not-a-date", start, end); err == nil {
		t.Error("Contains(\"not-a-date\") = nil error, want error")
	}
}

func TestDateOnly(t *testing.T) {
	withTime := time.Date(2026, 3, 18, 14, 30, 45, 123, time.UTC)
	got := DateOnly(withTime)
	want := mustDay(t, "2026-03-18")
	if !got.Equal(want) {
		t.Errorf("DateOnly = %s, want %s", got, want)
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
		t.Errorf("DateOnly did not zero the time component: %s", got)
	}
}
