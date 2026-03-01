package dosing

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"canonical code OD", "OD", false},
		{"canonical code BD", "BD", false},
		{"alias BID", "BID", false},
		{"alias lowercase", "twice daily", false},
		{"alias with dots", "b.d.", false},
		{"every N hours", "every 4 hours", false},
		{"empty input", "", true},
		{"unknown code", "XYZZY", true},
		{"garbage", "!!!??", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstruction_NilInstruction(t *testing.T) {
	warnings := ValidateInstruction(nil)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Level != "error" || warnings[0].Field != "instruction" {
		t.Errorf("unexpected warning: %+v", warnings[0])
	}
}

func TestValidateInstruction_NilFrequency(t *testing.T) {
	warnings := ValidateInstruction(&DosingInstruction{})
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Level != "error" || warnings[0].Field != "frequency" {
		t.Errorf("unexpected warning: %+v", warnings[0])
	}
}

func TestValidateInstruction_ValidSimple(t *testing.T) {
	fc, _ := Parse("BD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
	})
	// Only expected: no warnings (route present, dose valid, not PRN)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for valid instruction, got %d: %+v", len(warnings), warnings)
	}
}

func TestValidateInstruction_PRNWithoutMaxDose(t *testing.T) {
	fc, _ := Parse("PRN")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
	})
	found := findWarning(warnings, "max_dose", "warning")
	if !found {
		t.Error("expected warning for PRN without max dose")
	}
}

func TestValidateInstruction_PRNWithMaxDose(t *testing.T) {
	fc, _ := Parse("PRN")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
		MaxDose:   &MaxDose{MaxPerDay: floatPtr(2000), MaxPerDayUnit: "mg"},
	})
	found := findWarning(warnings, "max_dose", "warning")
	if found {
		t.Error("should not warn about max_dose when it is present")
	}
}

func TestValidateInstruction_MealModifierOnPRN(t *testing.T) {
	fc, _ := Parse("PRN")
	ac, _ := Parse("AC")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency:    fc,
		MealModifier: ac,
		Dose:         &Dose{Value: 500, Unit: "mg"},
		Route:        "PO",
		MaxDose:      &MaxDose{MaxPerDay: floatPtr(2000), MaxPerDayUnit: "mg"},
	})
	found := findWarning(warnings, "meal_modifier", "warning")
	if !found {
		t.Error("expected warning for meal modifier on PRN frequency")
	}
}

func TestValidateInstruction_MealModifierOnRegular(t *testing.T) {
	fc, _ := Parse("BD")
	ac, _ := Parse("AC")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency:    fc,
		MealModifier: ac,
		Dose:         &Dose{Value: 500, Unit: "mg"},
		Route:        "PO",
	})
	found := findWarning(warnings, "meal_modifier", "warning")
	if found {
		t.Error("should not warn about meal modifier on regular frequency")
	}
}

func TestValidateInstruction_DurationWithOneOff(t *testing.T) {
	fc, _ := Parse("STAT")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "IV",
		Duration:  &Duration{Value: 7, Unit: PeriodDay},
	})
	found := findWarning(warnings, "duration", "warning")
	if !found {
		t.Error("expected warning for duration with one-off frequency")
	}
}

func TestValidateInstruction_DoseZero(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 0, Unit: "mg"},
		Route:     "PO",
	})
	found := findWarning(warnings, "dose.value", "error")
	if !found {
		t.Error("expected error for zero dose value")
	}
}

func TestValidateInstruction_DoseNegative(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: -5, Unit: "mg"},
		Route:     "PO",
	})
	found := findWarning(warnings, "dose.value", "error")
	if !found {
		t.Error("expected error for negative dose value")
	}
}

func TestValidateInstruction_EmptyDoseUnit(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: ""},
		Route:     "PO",
	})
	found := findWarning(warnings, "dose.unit", "warning")
	if !found {
		t.Error("expected warning for empty dose unit")
	}
}

func TestValidateInstruction_DoseRangeInverted(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose: &Dose{
			Value:     0,
			Unit:      "tablets",
			LowValue:  floatPtr(3),
			HighValue: floatPtr(1),
		},
		Route: "PO",
	})
	found := findWarning(warnings, "dose.range", "error")
	if !found {
		t.Error("expected error for inverted dose range")
	}
}

func TestValidateInstruction_DoseRangeValid(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose: &Dose{
			Value:     0,
			Unit:      "tablets",
			LowValue:  floatPtr(1),
			HighValue: floatPtr(2),
		},
		Route: "PO",
	})
	found := findWarning(warnings, "dose.range", "error")
	if found {
		t.Error("should not error for valid dose range")
	}
}

func TestValidateInstruction_MissingRoute(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
	})
	found := findWarning(warnings, "route", "info")
	if !found {
		t.Error("expected info for missing route when dose is present")
	}
}

func TestValidateInstruction_NoRouteNoDose(t *testing.T) {
	fc, _ := Parse("OD")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
	})
	found := findWarning(warnings, "route", "info")
	if found {
		t.Error("should not warn about route when dose is absent")
	}
}

func TestValidateInstruction_DailyDoseExceedsMax(t *testing.T) {
	fc, _ := Parse("QDS") // frequency=4
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 600, Unit: "mg"},
		Route:     "PO",
		MaxDose:   &MaxDose{MaxPerDay: floatPtr(2000), MaxPerDayUnit: "mg"},
	})
	// 600 * 4 = 2400 > 2000
	found := findWarning(warnings, "dose", "warning")
	if !found {
		t.Error("expected warning for daily dose exceeding max")
	}
}

func TestValidateInstruction_DailyDoseWithinMax(t *testing.T) {
	fc, _ := Parse("QDS") // frequency=4
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
		MaxDose:   &MaxDose{MaxPerDay: floatPtr(2000), MaxPerDayUnit: "mg"},
	})
	// 500 * 4 = 2000, not > 2000
	found := findWarning(warnings, "dose", "warning")
	if found {
		t.Error("should not warn when daily dose equals max")
	}
}

func TestValidateInstruction_DailyDoseUnitMismatch(t *testing.T) {
	fc, _ := Parse("QDS")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 600, Unit: "mg"},
		Route:     "PO",
		MaxDose:   &MaxDose{MaxPerDay: floatPtr(2), MaxPerDayUnit: "g"},
	})
	// Units don't match — skip the check
	found := findWarning(warnings, "dose", "warning")
	if found {
		t.Error("should not warn when dose and max dose units differ")
	}
}

func TestValidateInstruction_DurationWithPRN(t *testing.T) {
	fc, _ := Parse("PRN")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
		Duration:  &Duration{Value: 7, Unit: PeriodDay},
		MaxDose:   &MaxDose{MaxPerDay: floatPtr(2000), MaxPerDayUnit: "mg"},
	})
	found := findWarning(warnings, "duration", "info")
	if !found {
		t.Error("expected info for duration with PRN frequency")
	}
}

func TestValidateInstruction_MultipleWarnings(t *testing.T) {
	fc, _ := Parse("PRN")
	warnings := ValidateInstruction(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: -1, Unit: ""},
		Duration:  &Duration{Value: 7, Unit: PeriodDay},
	})
	// Expect: PRN without max_dose (warning), dose<=0 (error), empty unit (warning),
	// missing route (info), duration+PRN (info)
	if len(warnings) < 4 {
		t.Errorf("expected at least 4 warnings, got %d: %+v", len(warnings), warnings)
	}
}

// findWarning checks if a warning with the given field and level exists.
func findWarning(warnings []ValidationWarning, field, level string) bool {
	for _, w := range warnings {
		if w.Field == field && w.Level == level {
			return true
		}
	}
	return false
}
