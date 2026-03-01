package dosing

import (
	"testing"
)

func TestToText(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		locale string
		want   string
	}{
		{"OD en-GB", "OD", LocaleEnGB, "Once daily"},
		{"OD en-US", "OD", LocaleEnUS, "Once daily"},
		{"BD en-GB", "BD", LocaleEnGB, "Twice daily"},
		{"BID alias resolves", "BID", LocaleEnUS, "Twice daily"},
		{"TDS en-GB", "TDS", LocaleEnGB, "Three times daily"},
		{"QDS en-GB", "QDS", LocaleEnGB, "Four times daily"},
		{"Q4H", "Q4H", LocaleEnGB, "Every 4 hours"},
		{"MANE", "MANE", LocaleEnGB, "In the morning"},
		{"NOCTE", "NOCTE", LocaleEnGB, "At night"},
		{"PRN", "PRN", LocaleEnGB, "As needed"},
		{"STAT", "STAT", LocaleEnGB, "Immediately"},
		{"QOD", "QOD", LocaleEnGB, "Every other day"},
		{"WEEKLY", "WEEKLY", LocaleEnGB, "Once weekly"},
		{"MONTHLY", "MONTHLY", LocaleEnGB, "Once monthly"},
		{"AC", "AC", LocaleEnGB, "Before meals"},
		{"unknown locale falls back", "OD", "fr", "Once daily"},
		{"every 6 hours alias", "every 6 hours", LocaleEnGB, "Every 6 hours"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToText(tt.code, tt.locale)
			if err != nil {
				t.Fatalf("ToText(%q, %q) error: %v", tt.code, tt.locale, err)
			}
			if got != tt.want {
				t.Errorf("ToText(%q, %q) = %q, want %q", tt.code, tt.locale, got, tt.want)
			}
		})
	}
}

func TestToText_Errors(t *testing.T) {
	_, err := ToText("", LocaleEnGB)
	if err == nil {
		t.Error("expected error for empty code")
	}

	_, err = ToText("XYZZY", LocaleEnGB)
	if err == nil {
		t.Error("expected error for unknown code")
	}
}

func TestToLabel(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		locale string
		want   string
	}{
		{"OD en-GB", "OD", LocaleEnGB, "OD"},
		{"OD en-US", "OD", LocaleEnUS, "QD"},
		{"BD en-GB", "BD", LocaleEnGB, "BD"},
		{"BD en-US", "BD", LocaleEnUS, "BID"},
		{"BID resolves to BD label", "BID", LocaleEnGB, "BD"},
		{"TDS en-GB", "TDS", LocaleEnGB, "TDS"},
		{"TDS en-US", "TDS", LocaleEnUS, "TID"},
		{"QDS en-GB", "QDS", LocaleEnGB, "QDS"},
		{"QDS en-US", "QDS", LocaleEnUS, "QID"},
		{"Q4H same both locales", "Q4H", LocaleEnGB, "Q4H"},
		{"PRN both locales", "PRN", LocaleEnGB, "PRN"},
		{"STAT both locales", "STAT", LocaleEnUS, "STAT"},
		{"unknown locale falls back", "BD", "sw", "BD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToLabel(tt.code, tt.locale)
			if err != nil {
				t.Fatalf("ToLabel(%q, %q) error: %v", tt.code, tt.locale, err)
			}
			if got != tt.want {
				t.Errorf("ToLabel(%q, %q) = %q, want %q", tt.code, tt.locale, got, tt.want)
			}
		})
	}
}

func TestToLabel_Errors(t *testing.T) {
	_, err := ToLabel("", LocaleEnGB)
	if err == nil {
		t.Error("expected error for empty code")
	}
}

func TestInstructionToText_SimpleOral(t *testing.T) {
	fc, _ := Parse("BD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "500mg twice daily by mouth"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_WithDuration(t *testing.T) {
	fc, _ := Parse("TDS")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 250, Unit: "mg"},
		Route:     "PO",
		Duration:  &Duration{Value: 7, Unit: PeriodDay},
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "250mg three times daily by mouth for 7 days"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_WithMealModifier(t *testing.T) {
	fc, _ := Parse("BD")
	ac, _ := Parse("AC")
	got, err := InstructionToText(&DosingInstruction{
		Frequency:    fc,
		MealModifier: ac,
		Dose:         &Dose{Value: 500, Unit: "mg"},
		Route:        "PO",
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "500mg twice daily by mouth, before meals"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_WithMaxDose(t *testing.T) {
	fc, _ := Parse("PRN_Q4H")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
		MaxDose:   &MaxDose{MaxPerDay: floatPtr(2000), MaxPerDayUnit: "mg"},
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "500mg as needed, max every 4 hours by mouth (max 2000mg/day)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_RangeDose(t *testing.T) {
	fc, _ := Parse("OD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose: &Dose{
			Unit:      "tablets",
			LowValue:  floatPtr(1),
			HighValue: floatPtr(2),
		},
		Route: "PO",
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "1-2 tablets once daily by mouth"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_DecimalDose(t *testing.T) {
	fc, _ := Parse("BD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 2.5, Unit: "ml"},
		Route:     "PO",
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "2.5ml twice daily by mouth"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_FrequencyOnly(t *testing.T) {
	fc, _ := Parse("OD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "Once daily"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_IVRoute(t *testing.T) {
	fc, _ := Parse("STAT")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 1000, Unit: "mg"},
		Route:     "IV",
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "1000mg immediately intravenously"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_DurationSingular(t *testing.T) {
	fc, _ := Parse("OD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
		Duration:  &Duration{Value: 1, Unit: PeriodWeek},
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "500mg once daily by mouth for 1 week"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_WithInstructions(t *testing.T) {
	fc, _ := Parse("OD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency:    fc,
		Dose:         &Dose{Value: 20, Unit: "mg"},
		Route:        "PO",
		Instructions: []string{"take with water"},
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "20mg once daily by mouth, take with water"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_MaxPerDose(t *testing.T) {
	fc, _ := Parse("PRN")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
		MaxDose:   &MaxDose{MaxPerDose: floatPtr(1000), MaxPerDoseUnit: "mg", MaxPerDay: floatPtr(4000), MaxPerDayUnit: "mg"},
	}, LocaleEnGB)
	if err != nil {
		t.Fatal(err)
	}
	want := "500mg as needed by mouth (max 1000mg/dose, max 4000mg/day)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInstructionToText_Errors(t *testing.T) {
	_, err := InstructionToText(nil, LocaleEnGB)
	if err == nil {
		t.Error("expected error for nil instruction")
	}

	_, err = InstructionToText(&DosingInstruction{}, LocaleEnGB)
	if err == nil {
		t.Error("expected error for nil frequency")
	}
}

func TestResolveLocale(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en-GB", "en-GB"},
		{"en-US", "en-US"},
		{"en_GB", "en-GB"},
		{"en_US", "en-US"},
		{"en-gb", "en-GB"},
		{"en-us", "en-US"},
		{"en_gb", "en-GB"},
		{"en_us", "en-US"},
		{"fr", "en-GB"},
		{"", "en-GB"},
		{"unknown", "en-GB"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveLocale(tt.input)
			if got != tt.want {
				t.Errorf("resolveLocale(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{500, "500"},
		{1, "1"},
		{0, "0"},
		{2.5, "2.5"},
		{0.25, "0.25"},
		{100.0, "100"},
	}

	for _, tt := range tests {
		got := formatFloat(tt.input)
		if got != tt.want {
			t.Errorf("formatFloat(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRouteToText(t *testing.T) {
	tests := []struct {
		route string
		want  string
	}{
		{"PO", "by mouth"},
		{"po", "by mouth"},
		{"IV", "intravenously"},
		{"IM", "intramuscularly"},
		{"SC", "subcutaneously"},
		{"SUBCUT", "subcutaneously"},
		{"SL", "sublingually"},
		{"PR", "rectally"},
		{"INH", "by inhalation"},
		{"TOP", "topically"},
		{"NAS", "nasally"},
		{"OPH", "into the eye"},
		{"OT", "into the ear"},
		{"PV", "vaginally"},
		{"NEB", "by nebuliser"},
		{"CUSTOM", "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {
			got := routeToText(tt.route)
			if got != tt.want {
				t.Errorf("routeToText(%q) = %q, want %q", tt.route, got, tt.want)
			}
		})
	}
}

func TestInstructionToText_EnUS(t *testing.T) {
	fc, _ := Parse("BD")
	got, err := InstructionToText(&DosingInstruction{
		Frequency: fc,
		Dose:      &Dose{Value: 500, Unit: "mg"},
		Route:     "PO",
	}, LocaleEnUS)
	if err != nil {
		t.Fatal(err)
	}
	want := "500mg twice daily by mouth"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
