package dosing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schedule-specific errors.
var (
	ErrPRNNoSchedule     = errors.New("PRN codes have no fixed schedule")
	ErrMealRelNoSchedule = errors.New("meal-relative codes require a base frequency for scheduling")
	ErrInvalidDays       = errors.New("days must be positive")
	ErrInvalidTimeFormat = errors.New("custom time must be HH:MM format")
)

// Schedule generates concrete administration times for the given frequency code,
// starting from start, spanning days calendar days. Day 1 skips times that have
// already passed relative to start.
func Schedule(code string, start time.Time, days int) ([]time.Time, error) {
	if days <= 0 {
		return nil, ErrInvalidDays
	}

	fc, err := Parse(code)
	if err != nil {
		return nil, err
	}

	return scheduleFromFC(fc, start, days, nil)
}

// ScheduleWithTimes is like Schedule but uses customTimes (HH:MM strings) instead
// of the frequency code's DefaultTimes. Rolling-interval codes ignore customTimes.
// PRN and pure meal modifier codes still return errors.
func ScheduleWithTimes(code string, start time.Time, days int, customTimes []string) ([]time.Time, error) {
	if days <= 0 {
		return nil, ErrInvalidDays
	}

	// Validate custom times format.
	for _, t := range customTimes {
		if _, _, err := parseTimeOfDay(t); err != nil {
			return nil, fmt.Errorf("%w: %q", ErrInvalidTimeFormat, t)
		}
	}

	fc, err := Parse(code)
	if err != nil {
		return nil, err
	}

	return scheduleFromFC(fc, start, days, customTimes)
}

// scheduleFromFC dispatches to the correct scheduling strategy based on category.
func scheduleFromFC(fc *FrequencyCode, start time.Time, days int, customTimes []string) ([]time.Time, error) {
	switch fc.Category {
	case CategoryPRN:
		return nil, ErrPRNNoSchedule

	case CategoryMealRelative:
		// AC_HS has Frequency > 0, so it can be scheduled with fixed times.
		if fc.Frequency > 0 && len(fc.DefaultTimes) > 0 {
			times := fc.DefaultTimes
			if customTimes != nil {
				times = customTimes
			}
			return generateFixedTimesSchedule(start, days, times), nil
		}
		// Pure modifiers (AC, PC, CC) cannot be scheduled.
		return nil, ErrMealRelNoSchedule

	case CategoryOneOff:
		return []time.Time{start}, nil

	case CategoryExtended:
		times := fc.DefaultTimes
		if customTimes != nil {
			times = customTimes
		}
		return generateExtendedSchedule(fc, start, days, times), nil

	case CategoryInterval:
		// Interval codes with empty DefaultTimes are truly rolling.
		if len(fc.DefaultTimes) == 0 {
			// Rolling intervals ignore customTimes.
			return generateRollingSchedule(start, days, fc.IntervalHours), nil
		}
		// Interval codes with DefaultTimes use fixed daily times.
		times := fc.DefaultTimes
		if customTimes != nil {
			times = customTimes
		}
		return generateFixedTimesSchedule(start, days, times), nil

	default:
		// CategoryRegular, CategoryTimeOfDay — fixed daily times.
		times := fc.DefaultTimes
		if customTimes != nil {
			times = customTimes
		}
		return generateFixedTimesSchedule(start, days, times), nil
	}
}

// generateFixedTimesSchedule generates times at the given HH:MM slots each day.
// On day 1, times before start are skipped.
func generateFixedTimesSchedule(start time.Time, days int, times []string) []time.Time {
	var result []time.Time
	loc := start.Location()

	for d := 0; d < days; d++ {
		dayStart := time.Date(start.Year(), start.Month(), start.Day()+d, 0, 0, 0, 0, loc)

		for _, t := range times {
			hour, min, err := parseTimeOfDay(t)
			if err != nil {
				continue
			}
			admin := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), hour, min, 0, 0, loc)

			// On day 1, skip times already passed.
			if d == 0 && admin.Before(start) {
				continue
			}
			result = append(result, admin)
		}
	}
	return result
}

// generateRollingSchedule generates times at regular intervals from start.
func generateRollingSchedule(start time.Time, days int, intervalHours float64) []time.Time {
	var result []time.Time
	interval := time.Duration(intervalHours * float64(time.Hour))
	end := time.Date(start.Year(), start.Month(), start.Day()+days, start.Hour(), start.Minute(), start.Second(), 0, start.Location())

	t := start
	for t.Before(end) {
		result = append(result, t)
		t = t.Add(interval)
	}
	return result
}

// generateExtendedSchedule generates times for QOD, WEEKLY, BIWEEKLY, MONTHLY.
func generateExtendedSchedule(fc *FrequencyCode, start time.Time, days int, times []string) []time.Time {
	var result []time.Time
	loc := start.Location()
	end := time.Date(start.Year(), start.Month(), start.Day()+days, 0, 0, 0, 0, loc)

	// Determine the time of day for each administration.
	hour, min := 8, 0 // default
	if len(times) > 0 {
		if h, m, err := parseTimeOfDay(times[0]); err == nil {
			hour, min = h, m
		}
	}

	current := time.Date(start.Year(), start.Month(), start.Day(), hour, min, 0, 0, loc)

	// If the first admin time is before start, it gets skipped on day 1.
	if current.Before(start) {
		current = advanceExtended(current, fc)
	}

	for current.Before(end) {
		result = append(result, current)
		current = advanceExtended(current, fc)
	}
	return result
}

// advanceExtended steps to the next administration based on the FC's period/unit.
func advanceExtended(t time.Time, fc *FrequencyCode) time.Time {
	switch fc.PeriodUnit {
	case PeriodDay:
		return t.AddDate(0, 0, fc.Period)
	case PeriodWeek:
		return t.AddDate(0, 0, fc.Period*7)
	case PeriodMonth:
		return t.AddDate(0, fc.Period, 0)
	default:
		// Fallback: use IntervalHours.
		return t.Add(time.Duration(fc.IntervalHours * float64(time.Hour)))
	}
}

// parseTimeOfDay parses "HH:MM" into hour and minute.
func parseTimeOfDay(s string) (int, int, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("%w: %q", ErrInvalidTimeFormat, s)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("%w: %q", ErrInvalidTimeFormat, s)
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("%w: %q", ErrInvalidTimeFormat, s)
	}
	return hour, min, nil
}
