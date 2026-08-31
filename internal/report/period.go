package report

import "time"

// PeriodToday returns the local calendar day containing now (inclusive bounds).
func PeriodToday(now time.Time) (from, to time.Time) {
	loc := now.Location()
	y, m, d := now.Date()
	from = time.Date(y, m, d, 0, 0, 0, 0, loc)
	to = time.Date(y, m, d, 23, 59, 59, 999999999, loc)
	return from, to
}

// PeriodWeek returns the last 7 days including today (inclusive bounds).
func PeriodWeek(now time.Time) (from, to time.Time) {
	loc := now.Location()
	y, m, d := now.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, -6)
	_, to = PeriodToday(now)
	return start, to
}

// PeriodMonth returns the calendar month containing now (inclusive bounds).
func PeriodMonth(now time.Time) (from, to time.Time) {
	loc := now.Location()
	y, m, _ := now.Date()
	from = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	to = time.Date(y, m+1, 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	return from, to
}
