package daterange

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	dateLayout  = "2006-01-02"
	monthLayout = "2006-01"
)

// names must stay in sync with the keywords handled by Resolve.
var names = []string{"today", "yesterday", "week", "lastweek", "month", "lastmonth", "year", "lastyear"}

func Names() []string {
	return slices.Clone(names)
}

func Resolve(input string, now time.Time) (time.Time, time.Time, error) {
	r := strings.TrimSpace(strings.ToLower(input))
	if r == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("range cannot be empty")
	}

	today := DateOnly(now)

	switch r {
	case "today":
		return today, today, nil
	case "yesterday":
		y := today.AddDate(0, 0, -1)
		return y, y, nil
	case "week":
		start := startOfWeek(today)
		end := start.AddDate(0, 0, 6)
		return start, end, nil
	case "lastweek":
		start := startOfWeek(today).AddDate(0, 0, -7)
		end := start.AddDate(0, 0, 6)
		return start, end, nil
	case "month":
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		end := start.AddDate(0, 1, -1)
		return start, end, nil
	case "lastmonth":
		thisMonthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		start := thisMonthStart.AddDate(0, -1, 0)
		end := thisMonthStart.AddDate(0, 0, -1)
		return start, end, nil
	case "year":
		start := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, today.Location())
		end := time.Date(today.Year(), time.December, 31, 0, 0, 0, 0, today.Location())
		return start, end, nil
	case "lastyear":
		y := today.Year() - 1
		start := time.Date(y, time.January, 1, 0, 0, 0, 0, today.Location())
		end := time.Date(y, time.December, 31, 0, 0, 0, 0, today.Location())
		return start, end, nil
	}

	parts := strings.Split(r, "..")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid range '%s'", input)
	}

	start, err := time.ParseInLocation(dateLayout, strings.TrimSpace(parts[0]), now.Location())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start date '%s'", parts[0])
	}
	end, err := time.ParseInLocation(dateLayout, strings.TrimSpace(parts[1]), now.Location())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end date '%s'", parts[1])
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end date is before start date")
	}

	return start, end, nil
}

func Contains(dateValue string, start, end time.Time) (bool, error) {
	txDate, err := time.ParseInLocation(dateLayout, dateValue, start.Location())
	if err != nil {
		return false, err
	}

	if txDate.Before(start) || txDate.After(end) {
		return false, nil
	}

	return true, nil
}

func ParseMonth(input string, now time.Time) (time.Time, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), nil
	}

	month, err := time.ParseInLocation(monthLayout, trimmed, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month format: %s", input)
	}

	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, now.Location()), nil
}

func FormatMonth(month time.Time) string {
	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location()).Format(monthLayout)
}

func MonthBounds(month time.Time) (time.Time, time.Time) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	end := start.AddDate(0, 1, -1)
	return start, end
}

func DateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}
