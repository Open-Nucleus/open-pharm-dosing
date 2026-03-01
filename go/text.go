package dosing

import (
	"fmt"
	"math"
	"strings"
)

// Locale constants.
const (
	LocaleEnGB = "en-GB"
	LocaleEnUS = "en-US"
)

// ToText returns the human-readable display text for a dosing frequency code.
// Uses Parse() so aliases (e.g. "BID") resolve correctly. Falls back to en-GB
// silently for unsupported locales — clinical safety requires always producing output.
func ToText(code string, locale string) (string, error) {
	fc, err := Parse(code)
	if err != nil {
		return "", err
	}
	locale = resolveLocale(locale)
	if text, ok := fc.Display[locale]; ok {
		return text, nil
	}
	if text, ok := fc.Display[LocaleEnGB]; ok {
		return text, nil
	}
	return fc.Code, nil
}

// ToLabel returns the locale-preferred short code label for a dosing frequency.
// For example, "BD" in en-GB vs "BID" in en-US.
func ToLabel(code string, locale string) (string, error) {
	fc, err := Parse(code)
	if err != nil {
		return "", err
	}
	locale = resolveLocale(locale)
	if label, ok := fc.LocalePreferred[locale]; ok {
		return label, nil
	}
	if label, ok := fc.LocalePreferred[LocaleEnGB]; ok {
		return label, nil
	}
	return fc.Code, nil
}

// InstructionToText converts a DosingInstruction to a human-readable string.
// Output pattern: "{dose} {frequency} {route} for {duration}, {meal modifier} (max {max})"
func InstructionToText(instruction *DosingInstruction, locale string) (string, error) {
	if instruction == nil {
		return "", fmt.Errorf("instruction is nil")
	}
	if instruction.Frequency == nil {
		return "", fmt.Errorf("frequency is required")
	}

	locale = resolveLocale(locale)
	var parts []string

	// Dose
	hasDose := false
	if instruction.Dose != nil {
		parts = append(parts, formatDose(instruction.Dose))
		hasDose = true
	}

	// Frequency text
	freqText := displayText(instruction.Frequency, locale)
	if hasDose {
		freqText = strings.ToLower(freqText)
	}
	parts = append(parts, freqText)

	// Route
	if instruction.Route != "" {
		parts = append(parts, routeToText(instruction.Route))
	}

	// Duration
	if instruction.Duration != nil {
		parts = append(parts, formatDuration(instruction.Duration))
	}

	result := strings.Join(parts, " ")

	// Meal modifier — appended with comma
	if instruction.MealModifier != nil {
		mealText := displayText(instruction.MealModifier, locale)
		result += ", " + strings.ToLower(mealText)
	}

	// Max dose
	if instruction.MaxDose != nil {
		maxParts := formatMaxDose(instruction.MaxDose)
		if maxParts != "" {
			result += " (" + maxParts + ")"
		}
	}

	// Additional instructions
	if len(instruction.Instructions) > 0 {
		result += ", " + strings.Join(instruction.Instructions, ", ")
	}

	return result, nil
}

// resolveLocale normalises and falls back to en-GB for unknown locales.
func resolveLocale(locale string) string {
	switch locale {
	case LocaleEnGB, LocaleEnUS:
		return locale
	case "en-gb", "en_GB", "en_gb":
		return LocaleEnGB
	case "en-us", "en_US", "en_us":
		return LocaleEnUS
	default:
		return LocaleEnGB
	}
}

// displayText returns the display text for a FrequencyCode in the given locale.
func displayText(fc *FrequencyCode, locale string) string {
	if text, ok := fc.Display[locale]; ok {
		return text
	}
	if text, ok := fc.Display[LocaleEnGB]; ok {
		return text
	}
	return fc.Code
}

// formatDose produces a dose string like "500mg", "2.5ml", or "1-2 tablets".
func formatDose(d *Dose) string {
	if d.LowValue != nil && d.HighValue != nil {
		return formatFloat(*d.LowValue) + "-" + formatFloat(*d.HighValue) + " " + d.Unit
	}
	return formatFloat(d.Value) + d.Unit
}

// formatFloat trims trailing ".0" from whole numbers.
func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%d", int(f))
	}
	return fmt.Sprintf("%g", f)
}

// formatDuration produces "for N units" text.
func formatDuration(d *Duration) string {
	unit := periodUnitToText(d.Unit, d.Value)
	return fmt.Sprintf("for %d %s", d.Value, unit)
}

// periodUnitToText converts a PeriodUnit to human text, pluralised as needed.
func periodUnitToText(u PeriodUnit, count int) string {
	switch u {
	case PeriodHour:
		if count == 1 {
			return "hour"
		}
		return "hours"
	case PeriodDay:
		if count == 1 {
			return "day"
		}
		return "days"
	case PeriodWeek:
		if count == 1 {
			return "week"
		}
		return "weeks"
	case PeriodMonth:
		if count == 1 {
			return "month"
		}
		return "months"
	default:
		return string(u)
	}
}

// routeToText maps route codes to human-readable text.
func routeToText(route string) string {
	switch strings.ToUpper(route) {
	case "PO":
		return "by mouth"
	case "IV":
		return "intravenously"
	case "IM":
		return "intramuscularly"
	case "SC", "SUBCUT":
		return "subcutaneously"
	case "SL":
		return "sublingually"
	case "PR":
		return "rectally"
	case "INH":
		return "by inhalation"
	case "TOP":
		return "topically"
	case "NAS":
		return "nasally"
	case "OPH":
		return "into the eye"
	case "OT":
		return "into the ear"
	case "PV":
		return "vaginally"
	case "NEB":
		return "by nebuliser"
	default:
		return route
	}
}

// formatMaxDose produces text like "max 2000mg/day" or "max 500mg/dose".
func formatMaxDose(m *MaxDose) string {
	var parts []string
	if m.MaxPerDose != nil {
		parts = append(parts, "max "+formatFloat(*m.MaxPerDose)+m.MaxPerDoseUnit+"/dose")
	}
	if m.MaxPerDay != nil {
		parts = append(parts, "max "+formatFloat(*m.MaxPerDay)+m.MaxPerDayUnit+"/day")
	}
	return strings.Join(parts, ", ")
}
