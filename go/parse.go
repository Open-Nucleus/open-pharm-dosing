package dosing

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var everyNHoursRe = regexp.MustCompile(`(?i)^every\s+(\d+)\s*h(?:ours?|rs?)?$`)

// Parse converts a free-text dosing frequency string into a FrequencyCode.
// It handles canonical codes, aliases, mixed case, punctuation variants,
// and "every N hours" patterns.
func Parse(input string) (*FrequencyCode, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrEmptyInput
	}

	// Try normalized alias lookup first.
	n := normalize(input)
	if fc, ok := aliasIndex[n]; ok {
		return fc, nil
	}

	// Try "every N hours" pattern.
	if fc := parseEveryNHours(input); fc != nil {
		return fc, nil
	}

	return nil, fmt.Errorf("%w: %q", ErrCodeNotFound, input)
}

// parseEveryNHours matches "every N hours" patterns and maps them to QNH codes.
func parseEveryNHours(input string) *FrequencyCode {
	m := everyNHoursRe.FindStringSubmatch(input)
	if m == nil {
		return nil
	}
	hours, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}
	code := fmt.Sprintf("Q%dH", hours)
	if fc, ok := registry[code]; ok {
		return fc
	}
	return nil
}

// ErrNotImplemented is returned by functions that are stubbed for future phases.
var ErrNotImplemented = errors.New("not implemented")

// ParseInstruction parses a full dosing instruction string into a DosingInstruction.
// This is a stub — full implementation deferred to Phase 4.
func ParseInstruction(input string) (*DosingInstruction, error) {
	return nil, fmt.Errorf("ParseInstruction: %w", ErrNotImplemented)
}
