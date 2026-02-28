package dosing

import (
	"errors"
	"testing"
)

func TestParseCanonicalCodes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"OD", "OD"},
		{"BD", "BD"},
		{"TDS", "TDS"},
		{"QDS", "QDS"},
		{"PRN", "PRN"},
		{"STAT", "STAT"},
		{"MANE", "MANE"},
		{"NOCTE", "NOCTE"},
		{"Q4H", "Q4H"},
		{"Q6H", "Q6H"},
		{"QOD", "QOD"},
		{"WEEKLY", "WEEKLY"},
		{"MONTHLY", "MONTHLY"},
	}

	for _, tt := range tests {
		fc, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if fc.Code != tt.want {
			t.Errorf("Parse(%q).Code = %q, want %q", tt.input, fc.Code, tt.want)
		}
	}
}

func TestParseAliasResolution(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BID", "BD"},
		{"TID", "TDS"},
		{"QID", "QDS"},
		{"QD", "OD"},
		{"HS", "NOCTE"},
		{"ON", "NOCTE"},
		{"AM", "MANE"},
		{"Q1W", "WEEKLY"},
		{"Q2W", "BIWEEKLY"},
		{"Q1M", "MONTHLY"},
		{"hourly", "Q1H"},
		{"fortnightly", "BIWEEKLY"},
		{"as needed", "PRN"},
		{"immediately", "STAT"},
		{"single dose", "ONCE"},
		{"before food", "AC"},
		{"after meals", "PC"},
		{"with food", "CC"},
	}

	for _, tt := range tests {
		fc, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if fc.Code != tt.want {
			t.Errorf("Parse(%q).Code = %q, want %q", tt.input, fc.Code, tt.want)
		}
	}
}

func TestParseMixedCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bd", "BD"},
		{"Bd", "BD"},
		{"bD", "BD"},
		{"BD", "BD"},
		{"tds", "TDS"},
		{"Tds", "TDS"},
		{"prn", "PRN"},
		{"Prn", "PRN"},
		{"stat", "STAT"},
		{"Stat", "STAT"},
		{"nocte", "NOCTE"},
		{"mane", "MANE"},
	}

	for _, tt := range tests {
		fc, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if fc.Code != tt.want {
			t.Errorf("Parse(%q).Code = %q, want %q", tt.input, fc.Code, tt.want)
		}
	}
}

func TestParsePunctuationVariants(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"b.d.", "BD"},
		{"b.i.d.", "BD"},
		{"t.d.s.", "TDS"},
		{"t.i.d.", "TDS"},
		{"q.d.s.", "QDS"},
		{"q.i.d.", "QDS"},
		{"p.r.n.", "PRN"},
		{"o.d.", "OD"},
		{"q.d.", "OD"},
		{"a.c.", "AC"},
		{"p.c.", "PC"},
		{"c.c.", "CC"},
		{"s.o.s.", "SOS"},
	}

	for _, tt := range tests {
		fc, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if fc.Code != tt.want {
			t.Errorf("Parse(%q).Code = %q, want %q", tt.input, fc.Code, tt.want)
		}
	}
}

func TestParseWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  BD  ", "BD"},
		{"\tTDS\t", "TDS"},
		{" PRN ", "PRN"},
		{"  once daily  ", "OD"},
	}

	for _, tt := range tests {
		fc, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if fc.Code != tt.want {
			t.Errorf("Parse(%q).Code = %q, want %q", tt.input, fc.Code, tt.want)
		}
	}
}

func TestParseEveryNHours(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"every 4 hours", "Q4H"},
		{"every 6 hours", "Q6H"},
		{"every 8 hours", "Q8H"},
		{"every 12 hours", "Q12H"},
		{"every 1 hour", "Q1H"},
		{"every 2 hours", "Q2H"},
		{"every 24 hours", "Q24H"},
		{"Every 4 Hours", "Q4H"},
		{"EVERY 6 HOURS", "Q6H"},
		{"every 4 hrs", "Q4H"},
		{"every 6 hr", "Q6H"},
	}

	for _, tt := range tests {
		fc, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if fc.Code != tt.want {
			t.Errorf("Parse(%q).Code = %q, want %q", tt.input, fc.Code, tt.want)
		}
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse("")
	if err != ErrEmptyInput {
		t.Errorf("Parse(\"\") error = %v, want ErrEmptyInput", err)
	}
}

func TestParseUnknown(t *testing.T) {
	_, err := Parse("INVALID_CODE_XYZ")
	if !errors.Is(err, ErrCodeNotFound) {
		t.Errorf("Parse(INVALID_CODE_XYZ) error = %v, want ErrCodeNotFound", err)
	}
}

func TestParseEveryNHoursUnknownInterval(t *testing.T) {
	_, err := Parse("every 5 hours")
	if !errors.Is(err, ErrCodeNotFound) {
		t.Errorf("Parse(\"every 5 hours\") error = %v, want ErrCodeNotFound", err)
	}
}

func TestParseInstructionStub(t *testing.T) {
	_, err := ParseInstruction("500mg BD")
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("ParseInstruction() error = %v, want ErrNotImplemented", err)
	}
}
