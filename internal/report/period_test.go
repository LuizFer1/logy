package report

import (
	"testing"
	"time"
)

func TestPeriodToday(t *testing.T) {
	loc := time.FixedZone("BRT", -3*3600)
	now := time.Date(2026, 8, 31, 15, 30, 0, 0, loc)

	from, to := PeriodToday(now)
	wantFrom := time.Date(2026, 8, 31, 0, 0, 0, 0, loc)
	wantTo := time.Date(2026, 8, 31, 23, 59, 59, 999999999, loc)

	if !from.Equal(wantFrom) {
		t.Fatalf("PeriodToday from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("PeriodToday to = %v, want %v", to, wantTo)
	}
}

func TestPeriodWeek(t *testing.T) {
	loc := time.FixedZone("BRT", -3*3600)
	now := time.Date(2026, 8, 31, 15, 30, 0, 0, loc)

	from, to := PeriodWeek(now)
	// last 7 days including today: Aug 25 00:00 → Aug 31 23:59:59.999999999
	wantFrom := time.Date(2026, 8, 25, 0, 0, 0, 0, loc)
	wantTo := time.Date(2026, 8, 31, 23, 59, 59, 999999999, loc)

	if !from.Equal(wantFrom) {
		t.Fatalf("PeriodWeek from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("PeriodWeek to = %v, want %v", to, wantTo)
	}
}

func TestPeriodMonth(t *testing.T) {
	loc := time.FixedZone("BRT", -3*3600)
	now := time.Date(2026, 8, 31, 15, 30, 0, 0, loc)

	from, to := PeriodMonth(now)
	wantFrom := time.Date(2026, 8, 1, 0, 0, 0, 0, loc)
	wantTo := time.Date(2026, 8, 31, 23, 59, 59, 999999999, loc)

	if !from.Equal(wantFrom) {
		t.Fatalf("PeriodMonth from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("PeriodMonth to = %v, want %v", to, wantTo)
	}
}

func TestPeriodMonthFebruary(t *testing.T) {
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	from, to := PeriodMonth(now)
	wantFrom := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 2, 28, 23, 59, 59, 999999999, time.UTC)

	if !from.Equal(wantFrom) {
		t.Fatalf("PeriodMonth from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("PeriodMonth to = %v, want %v", to, wantTo)
	}
}
