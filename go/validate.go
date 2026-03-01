package dosing

import "fmt"

// ValidationWarning represents a single finding from instruction validation.
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Level   string `json:"level"` // "error", "warning", "info"
}

// Validate checks whether the given code string resolves to a known frequency code.
func Validate(code string) error {
	if code == "" {
		return ErrEmptyInput
	}
	_, err := Parse(code)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	return nil
}

// ValidateInstruction performs clinical sense-checking on a DosingInstruction,
// returning multiple findings at different severity levels.
func ValidateInstruction(instruction *DosingInstruction) []ValidationWarning {
	if instruction == nil {
		return []ValidationWarning{{
			Field:   "instruction",
			Message: "instruction is nil",
			Level:   "error",
		}}
	}

	if instruction.Frequency == nil {
		return []ValidationWarning{{
			Field:   "frequency",
			Message: "frequency is required",
			Level:   "error",
		}}
	}

	var warnings []ValidationWarning
	fc := instruction.Frequency

	// Rule 3: PRN without MaxDose
	if fc.AsNeeded && instruction.MaxDose == nil {
		warnings = append(warnings, ValidationWarning{
			Field:   "max_dose",
			Message: "PRN frequency without maximum dose limit",
			Level:   "warning",
		})
	}

	// Rule 4: Meal modifier on incompatible category
	if instruction.MealModifier != nil {
		cat := fc.Category
		if cat != CategoryRegular && cat != CategoryInterval && cat != CategoryTimeOfDay {
			warnings = append(warnings, ValidationWarning{
				Field:   "meal_modifier",
				Message: "meal modifier on non-regular/interval/time-of-day frequency",
				Level:   "warning",
			})
		}
	}

	// Rule 5: Duration with one-off code
	if instruction.Duration != nil && fc.Category == CategoryOneOff {
		warnings = append(warnings, ValidationWarning{
			Field:   "duration",
			Message: "duration specified with one-off frequency",
			Level:   "warning",
		})
	}

	// Rule 6: Dose validation
	if instruction.Dose != nil {
		d := instruction.Dose

		// 6a: Value <= 0
		if d.Value <= 0 && d.LowValue == nil {
			warnings = append(warnings, ValidationWarning{
				Field:   "dose.value",
				Message: "dose value must be greater than zero",
				Level:   "error",
			})
		}

		// 6b: Empty unit
		if d.Unit == "" {
			warnings = append(warnings, ValidationWarning{
				Field:   "dose.unit",
				Message: "dose unit is empty",
				Level:   "warning",
			})
		}

		// 6c: Range dose LowValue > HighValue
		if d.LowValue != nil && d.HighValue != nil && *d.LowValue > *d.HighValue {
			warnings = append(warnings, ValidationWarning{
				Field:   "dose.range",
				Message: "dose range low value exceeds high value",
				Level:   "error",
			})
		}
	}

	// Rule 7: Missing route when dose is present
	if instruction.Dose != nil && instruction.Route == "" {
		warnings = append(warnings, ValidationWarning{
			Field:   "route",
			Message: "route not specified",
			Level:   "info",
		})
	}

	// Rule 8: Computed daily dose > MaxPerDay
	if instruction.Dose != nil && instruction.MaxDose != nil &&
		instruction.MaxDose.MaxPerDay != nil && fc.Frequency > 0 {
		dailyDose := instruction.Dose.Value * float64(fc.Frequency)
		if instruction.MaxDose.MaxPerDayUnit == instruction.Dose.Unit &&
			dailyDose > *instruction.MaxDose.MaxPerDay {
			warnings = append(warnings, ValidationWarning{
				Field:   "dose",
				Message: fmt.Sprintf("computed daily dose (%.4g) exceeds maximum (%.4g)", dailyDose, *instruction.MaxDose.MaxPerDay),
				Level:   "warning",
			})
		}
	}

	// Rule 9: Duration with PRN
	if instruction.Duration != nil && fc.AsNeeded {
		warnings = append(warnings, ValidationWarning{
			Field:   "duration",
			Message: "duration specified with PRN frequency is unusual",
			Level:   "info",
		})
	}

	return warnings
}
