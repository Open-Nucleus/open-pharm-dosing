package dosing

import (
	"encoding/json"
	"testing"
)

func TestToFhirTimingBD(t *testing.T) {
	data, err := ToFhirTiming("BD")
	if err != nil {
		t.Fatalf("ToFhirTiming(BD) error: %v", err)
	}

	var timing fhirTiming
	if err := json.Unmarshal(data, &timing); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if timing.Repeat == nil {
		t.Fatal("Repeat is nil")
	}
	if timing.Repeat.Frequency == nil || *timing.Repeat.Frequency != 2 {
		t.Errorf("Frequency = %v, want 2", timing.Repeat.Frequency)
	}
	if timing.Repeat.Period == nil || *timing.Repeat.Period != 1 {
		t.Errorf("Period = %v, want 1", timing.Repeat.Period)
	}
	if timing.Repeat.PeriodUnit == nil || *timing.Repeat.PeriodUnit != "d" {
		t.Errorf("PeriodUnit = %v, want d", timing.Repeat.PeriodUnit)
	}
	if len(timing.Repeat.TimeOfDay) != 2 {
		t.Errorf("TimeOfDay length = %d, want 2", len(timing.Repeat.TimeOfDay))
	}

	// Check FHIR code.
	if timing.Code == nil || len(timing.Code.Coding) == 0 {
		t.Fatal("FHIR code missing")
	}
	if timing.Code.Coding[0].Code != "BID" {
		t.Errorf("FHIR code = %q, want BID", timing.Code.Coding[0].Code)
	}
}

func TestToFhirTimingSTAT(t *testing.T) {
	data, err := ToFhirTiming("STAT")
	if err != nil {
		t.Fatalf("ToFhirTiming(STAT) error: %v", err)
	}

	var timing fhirTiming
	if err := json.Unmarshal(data, &timing); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if timing.Repeat == nil {
		t.Fatal("Repeat is nil")
	}
	if timing.Repeat.Count == nil || *timing.Repeat.Count != 1 {
		t.Errorf("Count = %v, want 1", timing.Repeat.Count)
	}
}

func TestToFhirTimingPRN(t *testing.T) {
	data, err := ToFhirTiming("PRN")
	if err != nil {
		t.Fatalf("ToFhirTiming(PRN) error: %v", err)
	}

	var timing fhirTiming
	if err := json.Unmarshal(data, &timing); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if timing.Repeat == nil {
		t.Fatal("Repeat is nil")
	}
	if timing.Repeat.AsNeeded == nil || !*timing.Repeat.AsNeeded {
		t.Error("AsNeeded should be true")
	}
}

func TestToFhirTimingNOCTE(t *testing.T) {
	data, err := ToFhirTiming("NOCTE")
	if err != nil {
		t.Fatalf("ToFhirTiming(NOCTE) error: %v", err)
	}

	var timing fhirTiming
	if err := json.Unmarshal(data, &timing); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if timing.Repeat == nil {
		t.Fatal("Repeat is nil")
	}
	if len(timing.Repeat.When) != 1 || timing.Repeat.When[0] != "NIGHT" {
		t.Errorf("When = %v, want [NIGHT]", timing.Repeat.When)
	}
}

func TestFromFhirTimingByCode(t *testing.T) {
	// BID FHIR code should resolve to BD.
	timing := fhirTiming{
		Repeat: &fhirTimingRepeat{
			Frequency:  intPtr(2),
			Period:     floatPtr(1),
			PeriodUnit: strPtr("d"),
		},
		Code: &fhirCodeableConcept{
			Coding: []fhirCoding{
				{System: fhirGTSSystem, Code: "BID"},
			},
		},
	}
	data, _ := json.Marshal(timing)
	fc, err := FromFhirTiming(data)
	if err != nil {
		t.Fatalf("FromFhirTiming error: %v", err)
	}
	if fc.Code != "BD" {
		t.Errorf("Code = %q, want BD", fc.Code)
	}
}

func TestFromFhirTimingByStructure(t *testing.T) {
	// period=2, periodUnit=d should resolve to QOD.
	timing := fhirTiming{
		Repeat: &fhirTimingRepeat{
			Frequency:  intPtr(1),
			Period:     floatPtr(2),
			PeriodUnit: strPtr("d"),
		},
	}
	data, _ := json.Marshal(timing)
	fc, err := FromFhirTiming(data)
	if err != nil {
		t.Fatalf("FromFhirTiming error: %v", err)
	}
	if fc.Code != "QOD" {
		t.Errorf("Code = %q, want QOD", fc.Code)
	}
}

// TestFhirRoundtripAllCodes is the key invariant test:
// FromFhirTiming(ToFhirTiming(code)) == code for all supported codes.
func TestFhirRoundtripAllCodes(t *testing.T) {
	// Some codes need special handling for roundtrip due to structural ambiguity.
	// Map codes to their expected roundtrip result.
	expectedRoundtrip := map[string]string{
		// ONCE and STAT both produce count=1; structure match defaults to STAT.
		"ONCE": "STAT",
		// AM_PM has same structure as BD (freq=2, period=1, d, same times).
		"AM_PM": "BD",
		// SOS is asNeeded without period; structure match defaults to PRN.
		"SOS": "PRN",
	}

	for _, code := range allCanonicalCodes {
		t.Run(code, func(t *testing.T) {
			// Convert to FHIR timing.
			data, err := ToFhirTiming(code)
			if err != nil {
				t.Fatalf("ToFhirTiming(%q) error: %v", code, err)
			}

			// Convert back.
			fc, err := FromFhirTiming(data)
			if err != nil {
				t.Fatalf("FromFhirTiming for %q error: %v", code, err)
			}

			expected := code
			if e, ok := expectedRoundtrip[code]; ok {
				expected = e
			}

			if fc.Code != expected {
				t.Errorf("Roundtrip(%q): got %q, want %q\nFHIR JSON: %s",
					code, fc.Code, expected, string(data))
			}
		})
	}
}

func TestToFhirDosageRoundtrip(t *testing.T) {
	doseVal := 500.0
	maxPerDay := 2000.0
	instruction := &DosingInstruction{
		Frequency: registry["BD"],
		Dose: &Dose{
			Value: doseVal,
			Unit:  "mg",
		},
		Route: "PO",
		MaxDose: &MaxDose{
			MaxPerDay:     &maxPerDay,
			MaxPerDayUnit: "mg",
		},
	}

	data, err := ToFhirDosage(instruction)
	if err != nil {
		t.Fatalf("ToFhirDosage error: %v", err)
	}

	result, err := FromFhirDosage(data)
	if err != nil {
		t.Fatalf("FromFhirDosage error: %v", err)
	}

	if result.Frequency == nil || result.Frequency.Code != "BD" {
		t.Errorf("Frequency = %v, want BD", result.Frequency)
	}
	if result.Route != "PO" {
		t.Errorf("Route = %q, want PO", result.Route)
	}
	if result.Dose == nil || result.Dose.Value != 500 || result.Dose.Unit != "mg" {
		t.Errorf("Dose = %v, want 500 mg", result.Dose)
	}
	if result.MaxDose == nil || result.MaxDose.MaxPerDay == nil || *result.MaxDose.MaxPerDay != 2000 {
		t.Errorf("MaxDose = %v, want 2000", result.MaxDose)
	}
}

func TestToFhirDosagePRN(t *testing.T) {
	instruction := &DosingInstruction{
		Frequency: registry["PRN"],
		Dose: &Dose{
			Value: 1,
			Unit:  "tablet",
		},
	}

	data, err := ToFhirDosage(instruction)
	if err != nil {
		t.Fatalf("ToFhirDosage error: %v", err)
	}

	// Verify asNeeded is at Dosage level.
	var dosage fhirDosage
	if err := json.Unmarshal(data, &dosage); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if dosage.AsNeededBool == nil || !*dosage.AsNeededBool {
		t.Error("Dosage.asNeededBoolean should be true")
	}

	// asNeeded should NOT be in timing.repeat for Dosage.
	if dosage.Timing != nil && dosage.Timing.Repeat != nil && dosage.Timing.Repeat.AsNeeded != nil {
		t.Error("Timing.repeat.asNeeded should be nil in Dosage (moved to Dosage level)")
	}

	// Roundtrip.
	result, err := FromFhirDosage(data)
	if err != nil {
		t.Fatalf("FromFhirDosage error: %v", err)
	}
	if result.Frequency == nil || result.Frequency.Code != "PRN" {
		t.Errorf("Frequency = %v, want PRN", result.Frequency)
	}
}

func TestToFhirTimingInvalidCode(t *testing.T) {
	_, err := ToFhirTiming("INVALID")
	if err == nil {
		t.Error("ToFhirTiming(INVALID) should return error")
	}
}

func TestFromFhirTimingInvalidJSON(t *testing.T) {
	_, err := FromFhirTiming([]byte("not json"))
	if err == nil {
		t.Error("FromFhirTiming(invalid) should return error")
	}
}

func TestToFhirDosageNil(t *testing.T) {
	_, err := ToFhirDosage(nil)
	if err == nil {
		t.Error("ToFhirDosage(nil) should return error")
	}
}
