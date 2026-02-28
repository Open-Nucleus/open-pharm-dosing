package dosing

import (
	"testing"
)

// All 31 canonical codes that must exist in the registry.
var allCanonicalCodes = []string{
	"OD", "BD", "TDS", "QDS", "5X_DAILY",
	"Q1H", "Q2H", "Q4H", "Q6H", "Q8H", "Q12H", "Q24H", "Q36H", "Q48H", "Q72H",
	"MANE", "NOCTE", "MIDI", "AM_PM",
	"AC", "PC", "CC", "AC_HS",
	"PRN", "PRN_Q4H", "PRN_Q6H", "SOS",
	"STAT", "ONCE",
	"QOD", "WEEKLY", "BIWEEKLY", "MONTHLY",
}

func TestRegistryCompleteness(t *testing.T) {
	for _, code := range allCanonicalCodes {
		fc, err := Get(code)
		if err != nil {
			t.Errorf("Get(%q) returned error: %v", code, err)
			continue
		}
		if fc.Code != code {
			t.Errorf("Get(%q).Code = %q, want %q", code, fc.Code, code)
		}
	}
}

func TestRegistryCount(t *testing.T) {
	codes := List()
	if len(codes) != 33 {
		t.Errorf("List() returned %d codes, want 33", len(codes))
	}
}

func TestGetKnownCode(t *testing.T) {
	fc, err := Get("BD")
	if err != nil {
		t.Fatalf("Get(BD) error: %v", err)
	}
	if fc.Code != "BD" {
		t.Errorf("Code = %q, want BD", fc.Code)
	}
	if fc.Frequency != 2 {
		t.Errorf("Frequency = %d, want 2", fc.Frequency)
	}
	if fc.Category != CategoryRegular {
		t.Errorf("Category = %q, want %q", fc.Category, CategoryRegular)
	}
	if fc.IntervalHours != 12 {
		t.Errorf("IntervalHours = %f, want 12", fc.IntervalHours)
	}
	if fc.Latin != "bis in die" {
		t.Errorf("Latin = %q, want %q", fc.Latin, "bis in die")
	}
	if fc.FhirCode != "BID" {
		t.Errorf("FhirCode = %q, want BID", fc.FhirCode)
	}
	if len(fc.DefaultTimes) != 2 {
		t.Errorf("DefaultTimes length = %d, want 2", len(fc.DefaultTimes))
	}
}

func TestGetUnknownCode(t *testing.T) {
	_, err := Get("INVALID")
	if err != ErrCodeNotFound {
		t.Errorf("Get(INVALID) error = %v, want ErrCodeNotFound", err)
	}
}

func TestGetEmptyCode(t *testing.T) {
	_, err := Get("")
	if err != ErrEmptyInput {
		t.Errorf("Get(\"\") error = %v, want ErrEmptyInput", err)
	}
}

func TestListSorted(t *testing.T) {
	codes := List()
	for i := 1; i < len(codes); i++ {
		if codes[i].SortOrder < codes[i-1].SortOrder {
			t.Errorf("List() not sorted: codes[%d].SortOrder=%d < codes[%d].SortOrder=%d",
				i, codes[i].SortOrder, i-1, codes[i-1].SortOrder)
		}
	}
}

func TestListWithCategory(t *testing.T) {
	tests := []struct {
		category Category
		want     int
	}{
		{CategoryRegular, 5},
		{CategoryInterval, 10},
		{CategoryTimeOfDay, 4},
		{CategoryMealRelative, 4},
		{CategoryPRN, 4},
		{CategoryOneOff, 2},
		{CategoryExtended, 4},
	}

	for _, tt := range tests {
		codes := List(WithCategory(tt.category))
		if len(codes) != tt.want {
			t.Errorf("List(WithCategory(%q)) returned %d codes, want %d",
				tt.category, len(codes), tt.want)
		}
		for _, fc := range codes {
			if fc.Category != tt.category {
				t.Errorf("List(WithCategory(%q)) returned code %q with category %q",
					tt.category, fc.Code, fc.Category)
			}
		}
	}
}

func TestSearchByCode(t *testing.T) {
	results := Search("BD")
	found := false
	for _, fc := range results {
		if fc.Code == "BD" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Search(BD) did not find BD")
	}
}

func TestSearchByAlias(t *testing.T) {
	results := Search("twice daily")
	found := false
	for _, fc := range results {
		if fc.Code == "BD" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Search(\"twice daily\") did not find BD")
	}
}

func TestSearchByDisplayText(t *testing.T) {
	results := Search("morning")
	found := false
	for _, fc := range results {
		if fc.Code == "MANE" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Search(\"morning\") did not find MANE")
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	r1 := Search("prn")
	r2 := Search("PRN")
	if len(r1) != len(r2) {
		t.Errorf("Search case sensitivity: prn=%d results, PRN=%d results", len(r1), len(r2))
	}
}

func TestSearchEmpty(t *testing.T) {
	results := Search("")
	if results != nil {
		t.Errorf("Search(\"\") returned %d results, want nil", len(results))
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"B.D.", "bd"},
		{"b.i.d.", "bid"},
		{"p.r.n.", "prn"},
		{"PRN Q4H", "prnq4h"},
		{"every 4 hours", "every4hours"},
		{"AC+HS", "ac+hs"},
		{"Q1W", "q1w"},
	}
	for _, tt := range tests {
		got := normalize(tt.input)
		if got != tt.want {
			t.Errorf("normalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
