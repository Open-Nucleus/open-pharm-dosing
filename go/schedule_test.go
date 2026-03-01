package dosing

import (
	"testing"
	"time"
)

// Helper to create a fixed start time in UTC: 2025-01-15 09:30:00
func testStart() time.Time {
	return time.Date(2025, 1, 15, 9, 30, 0, 0, time.UTC)
}

// Helper to create a time on a given day at HH:MM UTC relative to 2025-01-15.
func testTime(dayOffset, hour, min int) time.Time {
	return time.Date(2025, 1, 15+dayOffset, hour, min, 0, 0, time.UTC)
}

func TestSchedule_RegularBD(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("BD", start, 2)
	if err != nil {
		t.Fatal(err)
	}

	// Day 1: 08:00 skipped (before 09:30), 20:00 included
	// Day 2: 08:00, 20:00
	expected := []time.Time{
		testTime(0, 20, 0),
		testTime(1, 8, 0),
		testTime(1, 20, 0),
	}
	assertTimesEqual(t, "BD 2 days", expected, times)
}

func TestSchedule_RegularOD(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("OD", start, 3)
	if err != nil {
		t.Fatal(err)
	}

	// Day 1: 08:00 skipped (before 09:30)
	// Day 2: 08:00
	// Day 3: 08:00
	expected := []time.Time{
		testTime(1, 8, 0),
		testTime(2, 8, 0),
	}
	assertTimesEqual(t, "OD 3 days", expected, times)
}

func TestSchedule_RegularTDS(t *testing.T) {
	start := time.Date(2025, 1, 15, 6, 0, 0, 0, time.UTC)
	times, err := Schedule("TDS", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Start at 06:00, all three times are after: 08:00, 14:00, 20:00
	expected := []time.Time{
		testTime(0, 8, 0),
		testTime(0, 14, 0),
		testTime(0, 20, 0),
	}
	assertTimesEqual(t, "TDS 1 day from 06:00", expected, times)
}

func TestSchedule_RegularQDS(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("QDS", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	expected := []time.Time{
		testTime(0, 6, 0),
		testTime(0, 12, 0),
		testTime(0, 18, 0),
		testTime(0, 22, 0),
	}
	assertTimesEqual(t, "QDS 1 day from midnight", expected, times)
}

func TestSchedule_IntervalQ4H_FixedTimes(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("Q4H", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Q4H has DefaultTimes: 06:00, 10:00, 14:00, 18:00, 22:00, 02:00
	// Day 1 from 09:30: 10:00, 14:00, 18:00, 22:00, 02:00
	expected := []time.Time{
		testTime(0, 10, 0),
		testTime(0, 14, 0),
		testTime(0, 18, 0),
		testTime(0, 22, 0),
		testTime(0, 2, 0), // 02:00 is before 09:30 but on the same calendar day it's parsed as hour 2
	}

	// Actually: 02:00 on day 1 is BEFORE 09:30, so it gets skipped.
	expected = []time.Time{
		testTime(0, 10, 0),
		testTime(0, 14, 0),
		testTime(0, 18, 0),
		testTime(0, 22, 0),
	}
	assertTimesEqual(t, "Q4H 1 day from 09:30", expected, times)
}

func TestSchedule_IntervalQ6H(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("Q6H", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Q6H DefaultTimes: ["06:00", "12:00", "18:00", "00:00"] — order preserved.
	// From midnight: all included (00:00 == start, not before). Output follows array order.
	expected := []time.Time{
		testTime(0, 6, 0),
		testTime(0, 12, 0),
		testTime(0, 18, 0),
		testTime(0, 0, 0),
	}
	assertTimesEqual(t, "Q6H 1 day from midnight", expected, times)
}

func TestSchedule_IntervalQ1H_Rolling(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("Q1H", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Q1H has empty DefaultTimes → rolling from start.
	// 24 hours from 09:30: 09:30, 10:30, 11:30, ..., 08:30 next day
	// End is 2025-01-16 09:30. So 24 times total.
	if len(times) != 24 {
		t.Errorf("Q1H rolling 1 day: expected 24 times, got %d", len(times))
	}
	if !times[0].Equal(start) {
		t.Errorf("first time should be start, got %v", times[0])
	}
	if !times[1].Equal(start.Add(time.Hour)) {
		t.Errorf("second time should be start+1h, got %v", times[1])
	}
}

func TestSchedule_IntervalQ2H_Rolling(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("Q2H", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Rolling every 2h for 1 day = 12 times
	if len(times) != 12 {
		t.Errorf("Q2H rolling 1 day: expected 12 times, got %d", len(times))
	}
}

func TestSchedule_IntervalQ36H_Rolling(t *testing.T) {
	start := testStart()
	times, err := Schedule("Q36H", start, 4)
	if err != nil {
		t.Fatal(err)
	}

	// 4 days = 96h. At 36h intervals: 0, 36, 72 → 3 times (108h > 96h)
	// Actually end = start + 4 days exactly. 0h, 36h, 72h. Next would be 108h > 96h.
	if len(times) != 3 {
		t.Errorf("Q36H 4 days: expected 3 times, got %d", len(times))
	}
}

func TestSchedule_IntervalQ72H_Rolling(t *testing.T) {
	start := testStart()
	times, err := Schedule("Q72H", start, 7)
	if err != nil {
		t.Fatal(err)
	}

	// 7 days = 168h. At 72h intervals: 0, 72, 144 → 3 times (216h > 168h)
	if len(times) != 3 {
		t.Errorf("Q72H 7 days: expected 3 times, got %d", len(times))
	}
}

func TestSchedule_TimeOfDay_MANE(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("MANE", start, 3)
	if err != nil {
		t.Fatal(err)
	}

	// MANE DefaultTimes: 08:00
	// Day 1: 08:00 skipped (before 09:30)
	// Day 2: 08:00
	// Day 3: 08:00
	expected := []time.Time{
		testTime(1, 8, 0),
		testTime(2, 8, 0),
	}
	assertTimesEqual(t, "MANE 3 days", expected, times)
}

func TestSchedule_TimeOfDay_NOCTE(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("NOCTE", start, 2)
	if err != nil {
		t.Fatal(err)
	}

	// NOCTE DefaultTimes: 22:00
	// Day 1: 22:00 (after 09:30)
	// Day 2: 22:00
	expected := []time.Time{
		testTime(0, 22, 0),
		testTime(1, 22, 0),
	}
	assertTimesEqual(t, "NOCTE 2 days", expected, times)
}

func TestSchedule_TimeOfDay_AMPM(t *testing.T) {
	start := testStart() // 09:30
	times, err := Schedule("AM_PM", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// AM_PM DefaultTimes: 08:00, 20:00
	// Day 1: 08:00 skipped, 20:00
	expected := []time.Time{
		testTime(0, 20, 0),
	}
	assertTimesEqual(t, "AM_PM 1 day", expected, times)
}

func TestSchedule_OneOff_STAT(t *testing.T) {
	start := testStart()
	times, err := Schedule("STAT", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	expected := []time.Time{start}
	assertTimesEqual(t, "STAT", expected, times)
}

func TestSchedule_OneOff_ONCE(t *testing.T) {
	start := testStart()
	times, err := Schedule("ONCE", start, 5)
	if err != nil {
		t.Fatal(err)
	}

	// Always single time regardless of days
	expected := []time.Time{start}
	assertTimesEqual(t, "ONCE", expected, times)
}

func TestSchedule_PRN_Error(t *testing.T) {
	_, err := Schedule("PRN", testStart(), 1)
	if err != ErrPRNNoSchedule {
		t.Errorf("expected ErrPRNNoSchedule, got %v", err)
	}
}

func TestSchedule_PRNQ4H_Error(t *testing.T) {
	_, err := Schedule("PRN_Q4H", testStart(), 1)
	if err != ErrPRNNoSchedule {
		t.Errorf("expected ErrPRNNoSchedule, got %v", err)
	}
}

func TestSchedule_SOS_Error(t *testing.T) {
	_, err := Schedule("SOS", testStart(), 1)
	if err != ErrPRNNoSchedule {
		t.Errorf("expected ErrPRNNoSchedule, got %v", err)
	}
}

func TestSchedule_MealRelAC_Error(t *testing.T) {
	_, err := Schedule("AC", testStart(), 1)
	if err != ErrMealRelNoSchedule {
		t.Errorf("expected ErrMealRelNoSchedule, got %v", err)
	}
}

func TestSchedule_MealRelPC_Error(t *testing.T) {
	_, err := Schedule("PC", testStart(), 1)
	if err != ErrMealRelNoSchedule {
		t.Errorf("expected ErrMealRelNoSchedule, got %v", err)
	}
}

func TestSchedule_MealRelCC_Error(t *testing.T) {
	_, err := Schedule("CC", testStart(), 1)
	if err != ErrMealRelNoSchedule {
		t.Errorf("expected ErrMealRelNoSchedule, got %v", err)
	}
}

func TestSchedule_MealRelACHS_HasSchedule(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("AC_HS", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// AC_HS DefaultTimes: 07:30, 11:30, 17:30, 22:00
	expected := []time.Time{
		testTime(0, 7, 30),
		testTime(0, 11, 30),
		testTime(0, 17, 30),
		testTime(0, 22, 0),
	}
	assertTimesEqual(t, "AC_HS 1 day", expected, times)
}

func TestSchedule_ExtendedQOD(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("QOD", start, 7)
	if err != nil {
		t.Fatal(err)
	}

	// QOD: every 2 days, DefaultTimes: 08:00
	// Day 1(15th): 08:00, Day 3(17th): 08:00, Day 5(19th): 08:00, Day 7(21st): 08:00
	expected := []time.Time{
		time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 17, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 19, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 21, 8, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "QOD 7 days", expected, times)
}

func TestSchedule_ExtendedWeekly(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("WEEKLY", start, 21)
	if err != nil {
		t.Fatal(err)
	}

	// WEEKLY: every 7 days, DefaultTimes: 08:00
	expected := []time.Time{
		time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 22, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 29, 8, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "WEEKLY 21 days", expected, times)
}

func TestSchedule_ExtendedBiweekly(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("BIWEEKLY", start, 42)
	if err != nil {
		t.Fatal(err)
	}

	// BIWEEKLY: every 14 days
	expected := []time.Time{
		time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 29, 8, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "BIWEEKLY 42 days", expected, times)
}

func TestSchedule_ExtendedMonthly(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("MONTHLY", start, 90)
	if err != nil {
		t.Fatal(err)
	}

	// MONTHLY: AddDate(0,1,0)
	expected := []time.Time{
		time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "MONTHLY 90 days", expected, times)
}

func TestSchedule_MonthlyHandlesFeb(t *testing.T) {
	// Start Jan 31 — Go's AddDate(0,1,0) normalises Jan 31 + 1 month = Mar 3 in 2025
	// (31 days into February overflows). This is standard Go behavior.
	start := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	times, err := Schedule("MONTHLY", start, 90)
	if err != nil {
		t.Fatal(err)
	}

	if len(times) < 2 {
		t.Fatalf("expected at least 2 times, got %d", len(times))
	}
	// Verify monthly stepping works (exact dates depend on Go's AddDate normalisation)
	if times[0] != time.Date(2025, 1, 31, 8, 0, 0, 0, time.UTC) {
		t.Errorf("first time should be Jan 31, got %v", times[0])
	}
	// Jan 31 + 1 month = March 3 via Go's AddDate
	if times[1] != time.Date(2025, 3, 3, 8, 0, 0, 0, time.UTC) {
		t.Errorf("second time should be Mar 3, got %v", times[1])
	}
}

// Edge case: start after all daily times → 0 times on day 1
func TestSchedule_StartAfterAllTimes(t *testing.T) {
	start := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC)
	times, err := Schedule("BD", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// BD DefaultTimes: 08:00, 20:00 — both before 23:00
	if len(times) != 0 {
		t.Errorf("expected 0 times when start is after all daily times, got %d: %v", len(times), times)
	}
}

// Edge case: start exactly at a scheduled time → include it
func TestSchedule_StartExactlyAtTime(t *testing.T) {
	start := time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC)
	times, err := Schedule("OD", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	expected := []time.Time{
		time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "OD start exactly at 08:00", expected, times)
}

func TestSchedule_InvalidDays(t *testing.T) {
	_, err := Schedule("OD", testStart(), 0)
	if err != ErrInvalidDays {
		t.Errorf("expected ErrInvalidDays for days=0, got %v", err)
	}

	_, err = Schedule("OD", testStart(), -1)
	if err != ErrInvalidDays {
		t.Errorf("expected ErrInvalidDays for days=-1, got %v", err)
	}
}

func TestSchedule_InvalidCode(t *testing.T) {
	_, err := Schedule("XYZZY", testStart(), 1)
	if err == nil {
		t.Error("expected error for unknown code")
	}
}

func TestSchedule_EmptyCode(t *testing.T) {
	_, err := Schedule("", testStart(), 1)
	if err == nil {
		t.Error("expected error for empty code")
	}
}

// --- ScheduleWithTimes tests ---

func TestScheduleWithTimes_CustomOverride(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := ScheduleWithTimes("BD", start, 1, []string{"07:00", "19:00"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []time.Time{
		time.Date(2025, 1, 15, 7, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 19, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "BD custom times", expected, times)
}

func TestScheduleWithTimes_RollingIgnoresCustom(t *testing.T) {
	start := testStart()
	withCustom, err := ScheduleWithTimes("Q1H", start, 1, []string{"10:00", "14:00"})
	if err != nil {
		t.Fatal(err)
	}

	withDefault, err := Schedule("Q1H", start, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Rolling interval should ignore custom times.
	if len(withCustom) != len(withDefault) {
		t.Errorf("rolling should ignore custom times: custom=%d, default=%d", len(withCustom), len(withDefault))
	}
}

func TestScheduleWithTimes_PRNStillErrors(t *testing.T) {
	_, err := ScheduleWithTimes("PRN", testStart(), 1, []string{"08:00"})
	if err != ErrPRNNoSchedule {
		t.Errorf("expected ErrPRNNoSchedule, got %v", err)
	}
}

func TestScheduleWithTimes_InvalidTimeFormat(t *testing.T) {
	_, err := ScheduleWithTimes("OD", testStart(), 1, []string{"8am"})
	if err == nil {
		t.Error("expected error for invalid time format")
	}

	_, err = ScheduleWithTimes("OD", testStart(), 1, []string{"25:00"})
	if err == nil {
		t.Error("expected error for hour 25")
	}

	_, err = ScheduleWithTimes("OD", testStart(), 1, []string{"08:60"})
	if err == nil {
		t.Error("expected error for minute 60")
	}
}

func TestScheduleWithTimes_ExtendedCustomTime(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := ScheduleWithTimes("WEEKLY", start, 14, []string{"10:00"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []time.Time{
		time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 22, 10, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "WEEKLY custom 10:00", expected, times)
}

func TestScheduleWithTimes_InvalidDays(t *testing.T) {
	_, err := ScheduleWithTimes("OD", testStart(), 0, []string{"08:00"})
	if err != ErrInvalidDays {
		t.Errorf("expected ErrInvalidDays, got %v", err)
	}
}

func TestScheduleWithTimes_ACHS_CustomTimes(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	times, err := ScheduleWithTimes("AC_HS", start, 1, []string{"08:00", "12:00", "18:00", "22:00"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []time.Time{
		time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 18, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 22, 0, 0, 0, time.UTC),
	}
	assertTimesEqual(t, "AC_HS custom times", expected, times)
}

// --- parseTimeOfDay tests ---

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		input   string
		hour    int
		min     int
		wantErr bool
	}{
		{"08:00", 8, 0, false},
		{"00:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"12:30", 12, 30, false},
		{"02:00", 2, 0, false},
		{"24:00", 0, 0, true},
		{"08:60", 0, 0, true},
		{"-1:00", 0, 0, true},
		{"8am", 0, 0, true},
		{"", 0, 0, true},
		{"noon", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			h, m, err := parseTimeOfDay(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeOfDay(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err == nil {
				if h != tt.hour || m != tt.min {
					t.Errorf("parseTimeOfDay(%q) = %d:%d, want %d:%d", tt.input, h, m, tt.hour, tt.min)
				}
			}
		})
	}
}

// --- Alias/alias test ---

func TestSchedule_AliasResolution(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	// BID is alias for BD
	times, err := Schedule("BID", start, 1)
	if err != nil {
		t.Fatal(err)
	}
	expected := []time.Time{
		testTime(0, 8, 0),
		testTime(0, 20, 0),
	}
	assertTimesEqual(t, "BID alias", expected, times)
}

func TestSchedule_EveryNHoursAlias(t *testing.T) {
	start := testStart()
	times, err := Schedule("every 4 hours", start, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Q4H has fixed times — should resolve same as Q4H
	timesQ4H, _ := Schedule("Q4H", start, 1)
	if len(times) != len(timesQ4H) {
		t.Errorf("'every 4 hours' should resolve same as Q4H: got %d vs %d", len(times), len(timesQ4H))
	}
}

// --- Helper ---

func assertTimesEqual(t *testing.T, label string, expected, got []time.Time) {
	t.Helper()
	if len(expected) != len(got) {
		t.Errorf("%s: expected %d times, got %d", label, len(expected), len(got))
		for i, g := range got {
			t.Logf("  got[%d] = %v", i, g)
		}
		return
	}
	for i := range expected {
		if !expected[i].Equal(got[i]) {
			t.Errorf("%s: time[%d] expected %v, got %v", label, i, expected[i], got[i])
		}
	}
}
