package dosing

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Internal FHIR R4 types (unexported).

type fhirTiming struct {
	Repeat *fhirTimingRepeat    `json:"repeat,omitempty"`
	Code   *fhirCodeableConcept `json:"code,omitempty"`
}

type fhirTimingRepeat struct {
	Frequency  *int     `json:"frequency,omitempty"`
	Period     *float64 `json:"period,omitempty"`
	PeriodUnit *string  `json:"periodUnit,omitempty"`
	TimeOfDay  []string `json:"timeOfDay,omitempty"`
	When       []string `json:"when,omitempty"`
	Count      *int     `json:"count,omitempty"`
	AsNeeded   *bool    `json:"asNeeded,omitempty"`
}

type fhirCodeableConcept struct {
	Coding []fhirCoding `json:"coding,omitempty"`
	Text   string       `json:"text,omitempty"`
}

type fhirCoding struct {
	System string `json:"system,omitempty"`
	Code   string `json:"code,omitempty"`
}

type fhirDosage struct {
	Timing      *fhirTiming       `json:"timing,omitempty"`
	AsNeededBool *bool             `json:"asNeededBoolean,omitempty"`
	Route       *fhirCodeableConcept `json:"route,omitempty"`
	DoseAndRate []fhirDoseAndRate `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod *fhirRatio   `json:"maxDosePerPeriod,omitempty"`
	Text        string            `json:"text,omitempty"`
}

type fhirDoseAndRate struct {
	DoseQuantity *fhirQuantity `json:"doseQuantity,omitempty"`
}

type fhirQuantity struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type fhirRatio struct {
	Numerator   *fhirQuantity `json:"numerator,omitempty"`
	Denominator *fhirQuantity `json:"denominator,omitempty"`
}

// FHIR code index: maps FHIR GTS abbreviation code → FrequencyCode.
var fhirCodeIndex map[string]*FrequencyCode

func init() {
	fhirCodeIndex = make(map[string]*FrequencyCode)
	for _, fc := range allCodes {
		if fc.FhirCode != "" {
			fhirCodeIndex[fc.FhirCode] = fc
		}
	}
}

// Errors returned by FHIR operations.
var (
	ErrFhirConversion = errors.New("FHIR conversion error")
	ErrFhirNoMatch    = errors.New("no matching frequency code for FHIR timing")
)

// ToFhirTiming converts a canonical frequency code to a FHIR R4 Timing JSON.
func ToFhirTiming(code string) ([]byte, error) {
	fc, err := Get(code)
	if err != nil {
		// Try parsing as an alias.
		fc, err = Parse(code)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFhirConversion, err)
		}
	}

	timing := buildFhirTiming(fc)
	return json.Marshal(timing)
}

func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func buildFhirTiming(fc *FrequencyCode) *fhirTiming {
	t := &fhirTiming{
		Repeat: &fhirTimingRepeat{},
	}

	// Add FHIR code if available.
	if fc.FhirCode != "" {
		t.Code = &fhirCodeableConcept{
			Coding: []fhirCoding{
				{System: fc.FhirSystem, Code: fc.FhirCode},
			},
		}
	}

	switch fc.Category {
	case CategoryRegular:
		t.Repeat.Frequency = intPtr(fc.Frequency)
		t.Repeat.Period = floatPtr(float64(fc.Period))
		t.Repeat.PeriodUnit = strPtr(string(fc.PeriodUnit))
		if len(fc.DefaultTimes) > 0 {
			t.Repeat.TimeOfDay = fc.DefaultTimes
		}

	case CategoryInterval:
		t.Repeat.Frequency = intPtr(1)
		t.Repeat.Period = floatPtr(float64(fc.Period))
		t.Repeat.PeriodUnit = strPtr(string(fc.PeriodUnit))

	case CategoryTimeOfDay:
		t.Repeat.Frequency = intPtr(fc.Frequency)
		t.Repeat.Period = floatPtr(float64(fc.Period))
		t.Repeat.PeriodUnit = strPtr(string(fc.PeriodUnit))
		switch fc.Code {
		case "NOCTE":
			t.Repeat.When = []string{"NIGHT"}
		case "MANE":
			if len(fc.DefaultTimes) > 0 {
				t.Repeat.TimeOfDay = fc.DefaultTimes
			}
		case "MIDI":
			if len(fc.DefaultTimes) > 0 {
				t.Repeat.TimeOfDay = fc.DefaultTimes
			}
		case "AM_PM":
			if len(fc.DefaultTimes) > 0 {
				t.Repeat.TimeOfDay = fc.DefaultTimes
			}
		}

	case CategoryMealRelative:
		switch fc.Code {
		case "AC":
			t.Repeat.When = []string{"AC"}
		case "PC":
			t.Repeat.When = []string{"PC"}
		case "CC":
			t.Repeat.When = []string{"C"}
		case "AC_HS":
			t.Repeat.When = []string{"AC", "HS"}
			t.Repeat.Frequency = intPtr(fc.Frequency)
			t.Repeat.Period = floatPtr(float64(fc.Period))
			t.Repeat.PeriodUnit = strPtr(string(fc.PeriodUnit))
		}

	case CategoryPRN:
		t.Repeat.AsNeeded = boolPtr(true)
		if fc.Period > 0 {
			t.Repeat.Frequency = intPtr(1)
			t.Repeat.Period = floatPtr(float64(fc.Period))
			t.Repeat.PeriodUnit = strPtr(string(fc.PeriodUnit))
		}

	case CategoryOneOff:
		t.Repeat.Count = intPtr(1)

	case CategoryExtended:
		t.Repeat.Frequency = intPtr(fc.Frequency)
		t.Repeat.Period = floatPtr(float64(fc.Period))
		t.Repeat.PeriodUnit = strPtr(string(fc.PeriodUnit))
	}

	return t
}

// FromFhirTiming converts a FHIR R4 Timing JSON to a FrequencyCode.
// Uses three strategies: FHIR code match, when-based match, structure match.
func FromFhirTiming(timing []byte) (*FrequencyCode, error) {
	var t fhirTiming
	if err := json.Unmarshal(timing, &t); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON: %v", ErrFhirConversion, err)
	}

	// Strategy 1: FHIR code match.
	if t.Code != nil {
		for _, coding := range t.Code.Coding {
			if fc, ok := fhirCodeIndex[coding.Code]; ok {
				return fc, nil
			}
		}
	}

	if t.Repeat == nil {
		return nil, ErrFhirNoMatch
	}

	// Strategy 2: When-based match.
	if len(t.Repeat.When) > 0 {
		fc := matchByWhen(t.Repeat.When)
		if fc != nil {
			return fc, nil
		}
	}

	// Strategy 3: Structure match.
	fc := matchByStructure(t.Repeat)
	if fc != nil {
		return fc, nil
	}

	return nil, ErrFhirNoMatch
}

func matchByWhen(when []string) *FrequencyCode {
	if len(when) == 1 {
		switch when[0] {
		case "NIGHT":
			return registry["NOCTE"]
		case "MORN":
			return registry["MANE"]
		case "AC":
			return registry["AC"]
		case "PC":
			return registry["PC"]
		case "C":
			return registry["CC"]
		}
	}
	if len(when) == 2 {
		// Check for AC+HS.
		hasAC, hasHS := false, false
		for _, w := range when {
			if w == "AC" {
				hasAC = true
			}
			if w == "HS" {
				hasHS = true
			}
		}
		if hasAC && hasHS {
			return registry["AC_HS"]
		}
	}
	return nil
}

func matchByStructure(r *fhirTimingRepeat) *FrequencyCode {
	// Check for one-off (count=1).
	if r.Count != nil && *r.Count == 1 {
		// Distinguish STAT from ONCE: both are count=1.
		// Default to STAT if no other info.
		return registry["STAT"]
	}

	// Check for PRN.
	if r.AsNeeded != nil && *r.AsNeeded {
		// PRN with interval constraints.
		if r.Period != nil && r.PeriodUnit != nil && *r.PeriodUnit == "h" {
			period := int(*r.Period)
			switch period {
			case 4:
				return registry["PRN_Q4H"]
			case 6:
				return registry["PRN_Q6H"]
			}
		}
		// Check SOS: asNeeded without period info, but we default to PRN.
		return registry["PRN"]
	}

	// Match by frequency/period/periodUnit.
	if r.Frequency == nil || r.Period == nil || r.PeriodUnit == nil {
		return nil
	}
	freq := *r.Frequency
	period := *r.Period
	pu := *r.PeriodUnit

	// Interval-based (periodUnit = "h").
	if pu == "h" {
		p := int(period)
		code := fmt.Sprintf("Q%dH", p)
		if fc, ok := registry[code]; ok {
			return fc
		}
		return nil
	}

	// Daily regular.
	if pu == "d" && period == 1 {
		switch freq {
		case 1:
			// Could be OD, MANE, MIDI — disambiguate by timeOfDay.
			if len(r.TimeOfDay) == 1 {
				switch r.TimeOfDay[0] {
				case "12:00":
					return registry["MIDI"]
				case "08:00":
					// Could be OD or MANE. Check if FHIR code was already handled.
					// Default to OD.
					return registry["OD"]
				case "22:00":
					return registry["NOCTE"]
				}
			}
			return registry["OD"]
		case 2:
			// Could be BD or AM_PM — disambiguate by timeOfDay presence.
			if len(r.TimeOfDay) == 2 {
				// Both BD and AM_PM have same times. Default to BD (regular).
				return registry["BD"]
			}
			return registry["BD"]
		case 3:
			return registry["TDS"]
		case 4:
			return registry["QDS"]
		case 5:
			return registry["5X_DAILY"]
		}
	}

	// Extended intervals.
	if pu == "d" && period == 2 && freq == 1 {
		return registry["QOD"]
	}
	if pu == "wk" {
		switch {
		case period == 1 && freq == 1:
			return registry["WEEKLY"]
		case period == 2 && freq == 1:
			return registry["BIWEEKLY"]
		}
	}
	if pu == "mo" && period == 1 && freq == 1 {
		return registry["MONTHLY"]
	}

	return nil
}

// ToFhirDosage converts a DosingInstruction to FHIR R4 Dosage JSON.
func ToFhirDosage(instruction *DosingInstruction) ([]byte, error) {
	if instruction == nil || instruction.Frequency == nil {
		return nil, fmt.Errorf("%w: nil instruction or frequency", ErrFhirConversion)
	}

	timing := buildFhirTiming(instruction.Frequency)
	// For Dosage, asNeeded is at the Dosage level, not repeat level.
	asNeeded := false
	if timing.Repeat != nil && timing.Repeat.AsNeeded != nil && *timing.Repeat.AsNeeded {
		asNeeded = true
		timing.Repeat.AsNeeded = nil
	}

	dosage := &fhirDosage{
		Timing: timing,
	}

	if asNeeded {
		dosage.AsNeededBool = boolPtr(true)
	}

	if instruction.Route != "" {
		dosage.Route = &fhirCodeableConcept{
			Text: instruction.Route,
		}
	}

	if instruction.Dose != nil {
		dosage.DoseAndRate = []fhirDoseAndRate{
			{
				DoseQuantity: &fhirQuantity{
					Value: instruction.Dose.Value,
					Unit:  instruction.Dose.Unit,
				},
			},
		}
	}

	if instruction.MaxDose != nil && instruction.MaxDose.MaxPerDay != nil {
		dosage.MaxDosePerPeriod = &fhirRatio{
			Numerator: &fhirQuantity{
				Value: *instruction.MaxDose.MaxPerDay,
				Unit:  instruction.MaxDose.MaxPerDayUnit,
			},
			Denominator: &fhirQuantity{
				Value: 1,
				Unit:  "d",
			},
		}
	}

	return json.Marshal(dosage)
}

// FromFhirDosage converts FHIR R4 Dosage JSON to a DosingInstruction.
func FromFhirDosage(dosage []byte) (*DosingInstruction, error) {
	var d fhirDosage
	if err := json.Unmarshal(dosage, &d); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON: %v", ErrFhirConversion, err)
	}

	instruction := &DosingInstruction{}

	if d.Timing != nil {
		// If asNeeded is at the Dosage level, propagate it into timing for matching.
		if d.AsNeededBool != nil && *d.AsNeededBool {
			if d.Timing.Repeat == nil {
				d.Timing.Repeat = &fhirTimingRepeat{}
			}
			d.Timing.Repeat.AsNeeded = boolPtr(true)
		}

		timingJSON, err := json.Marshal(d.Timing)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFhirConversion, err)
		}
		fc, err := FromFhirTiming(timingJSON)
		if err != nil {
			return nil, err
		}
		instruction.Frequency = fc
	}

	if d.Route != nil {
		instruction.Route = d.Route.Text
	}

	if len(d.DoseAndRate) > 0 && d.DoseAndRate[0].DoseQuantity != nil {
		instruction.Dose = &Dose{
			Value: d.DoseAndRate[0].DoseQuantity.Value,
			Unit:  d.DoseAndRate[0].DoseQuantity.Unit,
		}
	}

	if d.MaxDosePerPeriod != nil && d.MaxDosePerPeriod.Numerator != nil {
		v := d.MaxDosePerPeriod.Numerator.Value
		instruction.MaxDose = &MaxDose{
			MaxPerDay:     &v,
			MaxPerDayUnit: d.MaxDosePerPeriod.Numerator.Unit,
		}
	}

	return instruction, nil
}
