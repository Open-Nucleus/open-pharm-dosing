// Package dosing provides structured medication dosing frequency encoding,
// parsing, and FHIR R4 Timing conversion.
package dosing

import (
	"errors"
	"sort"
	"strings"
)

// Category classifies a dosing frequency code.
type Category string

const (
	CategoryRegular      Category = "regular"
	CategoryInterval     Category = "interval"
	CategoryTimeOfDay    Category = "time_of_day"
	CategoryMealRelative Category = "meal_relative"
	CategoryPRN          Category = "prn"
	CategoryOneOff       Category = "one_off"
	CategoryExtended     Category = "extended"
)

// PeriodUnit represents the unit of the dosing period.
type PeriodUnit string

const (
	PeriodHour  PeriodUnit = "h"
	PeriodDay   PeriodUnit = "d"
	PeriodWeek  PeriodUnit = "wk"
	PeriodMonth PeriodUnit = "mo"
)

// MealRelation describes the relationship of dosing to meals.
type MealRelation string

const (
	MealNone   MealRelation = "none"
	MealBefore MealRelation = "before"
	MealAfter  MealRelation = "after"
	MealWith   MealRelation = "with"
)

// FrequencyCode represents a single dosing frequency with all its metadata.
type FrequencyCode struct {
	Code            string            `json:"code"`
	Aliases         []string          `json:"aliases"`
	LocalePreferred map[string]string `json:"locale_preferred"`
	Display         map[string]string `json:"display"`
	Category        Category          `json:"category"`
	Frequency       int               `json:"frequency"`
	Period          int               `json:"period"`
	PeriodUnit      PeriodUnit        `json:"period_unit"`
	IntervalHours   float64           `json:"interval_hours"`
	DefaultTimes    []string          `json:"default_times"`
	WakingOnly      bool              `json:"waking_only"`
	AsNeeded        bool              `json:"as_needed"`
	MaxPerDay       int               `json:"max_per_day"`
	MinInterval     *float64          `json:"min_interval"`
	MealRelation    MealRelation      `json:"meal_relation"`
	Latin           string            `json:"latin"`
	FhirCode        string            `json:"fhir_code"`
	FhirSystem      string            `json:"fhir_system"`
	SortOrder       int               `json:"sort_order"`
}

// DosingInstruction represents a complete dosing instruction combining
// frequency, dose, route, and modifiers.
type DosingInstruction struct {
	Frequency    *FrequencyCode `json:"frequency"`
	MealModifier *FrequencyCode `json:"meal_modifier,omitempty"`
	Dose         *Dose          `json:"dose,omitempty"`
	Route        string         `json:"route,omitempty"`
	Duration     *Duration      `json:"duration,omitempty"`
	MaxDose      *MaxDose       `json:"max_dose,omitempty"`
	Instructions []string       `json:"instructions,omitempty"`
}

// Dose represents a medication dose amount.
type Dose struct {
	Value     float64  `json:"value"`
	Unit      string   `json:"unit"`
	LowValue  *float64 `json:"low_value,omitempty"`
	HighValue *float64 `json:"high_value,omitempty"`
}

// Duration represents a time duration for a prescription.
type Duration struct {
	Value int        `json:"value"`
	Unit  PeriodUnit `json:"unit"`
}

// MaxDose represents maximum dosing constraints.
type MaxDose struct {
	MaxPerDose     *float64 `json:"max_per_dose,omitempty"`
	MaxPerDay      *float64 `json:"max_per_day,omitempty"`
	MaxPerDoseUnit string   `json:"max_per_dose_unit,omitempty"`
	MaxPerDayUnit  string   `json:"max_per_day_unit,omitempty"`
}

// ListOption configures List() behavior.
type ListOption func(*listConfig)

type listConfig struct {
	category *Category
}

// WithCategory filters List() results to a specific category.
func WithCategory(c Category) ListOption {
	return func(cfg *listConfig) {
		cfg.category = &c
	}
}

// Errors returned by registry operations.
var (
	ErrCodeNotFound = errors.New("frequency code not found")
	ErrEmptyInput   = errors.New("empty input")
)

// Package-level indexes built once at init.
var (
	registry   map[string]*FrequencyCode // canonical code → FrequencyCode
	aliasIndex map[string]*FrequencyCode // normalized alias → FrequencyCode
	allCodes   []*FrequencyCode          // sorted by SortOrder
)

func init() {
	codes := buildRegistry()
	registry = make(map[string]*FrequencyCode, len(codes))
	aliasIndex = make(map[string]*FrequencyCode, len(codes)*5)
	allCodes = codes

	sort.Slice(allCodes, func(i, j int) bool {
		return allCodes[i].SortOrder < allCodes[j].SortOrder
	})

	for _, fc := range codes {
		registry[fc.Code] = fc
		aliasIndex[normalize(fc.Code)] = fc
		for _, alias := range fc.Aliases {
			aliasIndex[normalize(alias)] = fc
		}
		// Also index locale-preferred codes.
		for _, lp := range fc.LocalePreferred {
			aliasIndex[normalize(lp)] = fc
		}
	}
}

// Get returns the FrequencyCode for the given canonical code.
func Get(code string) (*FrequencyCode, error) {
	if code == "" {
		return nil, ErrEmptyInput
	}
	fc, ok := registry[code]
	if !ok {
		return nil, ErrCodeNotFound
	}
	return fc, nil
}

// List returns all frequency codes, optionally filtered by category.
func List(opts ...ListOption) []*FrequencyCode {
	cfg := &listConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.category == nil {
		result := make([]*FrequencyCode, len(allCodes))
		copy(result, allCodes)
		return result
	}
	var result []*FrequencyCode
	for _, fc := range allCodes {
		if fc.Category == *cfg.category {
			result = append(result, fc)
		}
	}
	return result
}

// Search returns frequency codes matching the query as a substring of the
// code, any alias, or any display text. The search is case-insensitive.
func Search(query string) []*FrequencyCode {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var result []*FrequencyCode
	seen := make(map[string]bool)
	for _, fc := range allCodes {
		if seen[fc.Code] {
			continue
		}
		if matchesQuery(fc, q) {
			result = append(result, fc)
			seen[fc.Code] = true
		}
	}
	return result
}

func matchesQuery(fc *FrequencyCode, q string) bool {
	if strings.Contains(strings.ToLower(fc.Code), q) {
		return true
	}
	for _, alias := range fc.Aliases {
		if strings.Contains(strings.ToLower(alias), q) {
			return true
		}
	}
	for _, display := range fc.Display {
		if strings.Contains(strings.ToLower(display), q) {
			return true
		}
	}
	return false
}

// normalize lowercases and strips . - / and spaces for alias matching.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		switch r {
		case '.', ' ', '-', '/':
			return -1
		}
		return r
	}, s)
	return s
}

const fhirGTSSystem = "http://terminology.hl7.org/CodeSystem/v3-GTSAbbreviation"

func floatPtr(f float64) *float64 {
	return &f
}

// buildRegistry returns the complete embedded frequency registry.
func buildRegistry() []*FrequencyCode {
	return []*FrequencyCode{
		{
			Code:            "OD",
			Aliases:         []string{"QD", "od", "o.d.", "q.d.", "once daily", "daily"},
			LocalePreferred: map[string]string{"en-GB": "OD", "en-US": "QD"},
			Display:         map[string]string{"en-GB": "Once daily", "en-US": "Once daily"},
			Category:        CategoryRegular,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   24,
			DefaultTimes:    []string{"08:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			Latin:           "omni die",
			FhirCode:        "QD",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       1,
		},
		{
			Code:            "BD",
			Aliases:         []string{"BID", "bd", "b.d.", "b.i.d.", "twice daily"},
			LocalePreferred: map[string]string{"en-GB": "BD", "en-US": "BID"},
			Display:         map[string]string{"en-GB": "Twice daily", "en-US": "Twice daily"},
			Category:        CategoryRegular,
			Frequency:       2,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   12,
			DefaultTimes:    []string{"08:00", "20:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			Latin:           "bis in die",
			FhirCode:        "BID",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       2,
		},
		{
			Code:            "TDS",
			Aliases:         []string{"TID", "tds", "t.d.s.", "t.i.d.", "three times daily"},
			LocalePreferred: map[string]string{"en-GB": "TDS", "en-US": "TID"},
			Display:         map[string]string{"en-GB": "Three times daily", "en-US": "Three times daily"},
			Category:        CategoryRegular,
			Frequency:       3,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   8,
			DefaultTimes:    []string{"08:00", "14:00", "20:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			Latin:           "ter die sumendus",
			FhirCode:        "TID",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       3,
		},
		{
			Code:            "QDS",
			Aliases:         []string{"QID", "qds", "q.d.s.", "q.i.d.", "four times daily"},
			LocalePreferred: map[string]string{"en-GB": "QDS", "en-US": "QID"},
			Display:         map[string]string{"en-GB": "Four times daily", "en-US": "Four times daily"},
			Category:        CategoryRegular,
			Frequency:       4,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   6,
			DefaultTimes:    []string{"06:00", "12:00", "18:00", "22:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			Latin:           "quater die sumendus",
			FhirCode:        "QID",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       4,
		},
		{
			Code:            "5X_DAILY",
			Aliases:         []string{"5x daily", "five times daily"},
			LocalePreferred: map[string]string{"en-GB": "5x daily", "en-US": "5x daily"},
			Display:         map[string]string{"en-GB": "Five times daily", "en-US": "Five times daily"},
			Category:        CategoryRegular,
			Frequency:       5,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   4.8,
			DefaultTimes:    []string{"06:00", "10:00", "14:00", "18:00", "22:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       5,
		},
		// Interval-based frequencies
		{
			Code:            "Q1H",
			Aliases:         []string{"every 1 hour", "hourly", "q1h"},
			LocalePreferred: map[string]string{"en-GB": "Q1H", "en-US": "Q1H"},
			Display:         map[string]string{"en-GB": "Every 1 hour", "en-US": "Every 1 hour"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodHour,
			IntervalHours:   1,
			DefaultTimes:    []string{},
			MealRelation:    MealNone,
			FhirCode:        "Q1H",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       6,
		},
		{
			Code:            "Q2H",
			Aliases:         []string{"every 2 hours", "q2h"},
			LocalePreferred: map[string]string{"en-GB": "Q2H", "en-US": "Q2H"},
			Display:         map[string]string{"en-GB": "Every 2 hours", "en-US": "Every 2 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          2,
			PeriodUnit:      PeriodHour,
			IntervalHours:   2,
			DefaultTimes:    []string{},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       7,
		},
		{
			Code:            "Q4H",
			Aliases:         []string{"every 4 hours", "q4h"},
			LocalePreferred: map[string]string{"en-GB": "Q4H", "en-US": "Q4H"},
			Display:         map[string]string{"en-GB": "Every 4 hours", "en-US": "Every 4 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          4,
			PeriodUnit:      PeriodHour,
			IntervalHours:   4,
			DefaultTimes:    []string{"06:00", "10:00", "14:00", "18:00", "22:00", "02:00"},
			MealRelation:    MealNone,
			FhirCode:        "Q4H",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       8,
		},
		{
			Code:            "Q6H",
			Aliases:         []string{"every 6 hours", "q6h"},
			LocalePreferred: map[string]string{"en-GB": "Q6H", "en-US": "Q6H"},
			Display:         map[string]string{"en-GB": "Every 6 hours", "en-US": "Every 6 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          6,
			PeriodUnit:      PeriodHour,
			IntervalHours:   6,
			DefaultTimes:    []string{"06:00", "12:00", "18:00", "00:00"},
			MealRelation:    MealNone,
			FhirCode:        "Q6H",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       9,
		},
		{
			Code:            "Q8H",
			Aliases:         []string{"every 8 hours", "q8h"},
			LocalePreferred: map[string]string{"en-GB": "Q8H", "en-US": "Q8H"},
			Display:         map[string]string{"en-GB": "Every 8 hours", "en-US": "Every 8 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          8,
			PeriodUnit:      PeriodHour,
			IntervalHours:   8,
			DefaultTimes:    []string{"06:00", "14:00", "22:00"},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       10,
		},
		{
			Code:            "Q12H",
			Aliases:         []string{"every 12 hours", "q12h"},
			LocalePreferred: map[string]string{"en-GB": "Q12H", "en-US": "Q12H"},
			Display:         map[string]string{"en-GB": "Every 12 hours", "en-US": "Every 12 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          12,
			PeriodUnit:      PeriodHour,
			IntervalHours:   12,
			DefaultTimes:    []string{"08:00", "20:00"},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       11,
		},
		{
			Code:            "Q24H",
			Aliases:         []string{"every 24 hours", "q24h"},
			LocalePreferred: map[string]string{"en-GB": "Q24H", "en-US": "Q24H"},
			Display:         map[string]string{"en-GB": "Every 24 hours", "en-US": "Every 24 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          24,
			PeriodUnit:      PeriodHour,
			IntervalHours:   24,
			DefaultTimes:    []string{"08:00"},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       12,
		},
		{
			Code:            "Q36H",
			Aliases:         []string{"every 36 hours", "q36h"},
			LocalePreferred: map[string]string{"en-GB": "Q36H", "en-US": "Q36H"},
			Display:         map[string]string{"en-GB": "Every 36 hours", "en-US": "Every 36 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          36,
			PeriodUnit:      PeriodHour,
			IntervalHours:   36,
			DefaultTimes:    []string{},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       13,
		},
		{
			Code:            "Q48H",
			Aliases:         []string{"every 48 hours", "alternate days", "q48h"},
			LocalePreferred: map[string]string{"en-GB": "Q48H", "en-US": "Q48H"},
			Display:         map[string]string{"en-GB": "Every 48 hours", "en-US": "Every 48 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          48,
			PeriodUnit:      PeriodHour,
			IntervalHours:   48,
			DefaultTimes:    []string{"08:00"},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       14,
		},
		{
			Code:            "Q72H",
			Aliases:         []string{"every 72 hours", "q72h"},
			LocalePreferred: map[string]string{"en-GB": "Q72H", "en-US": "Q72H"},
			Display:         map[string]string{"en-GB": "Every 72 hours", "en-US": "Every 72 hours"},
			Category:        CategoryInterval,
			Frequency:       1,
			Period:          72,
			PeriodUnit:      PeriodHour,
			IntervalHours:   72,
			DefaultTimes:    []string{},
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       15,
		},
		// Time-of-day frequencies
		{
			Code:            "MANE",
			Aliases:         []string{"mane", "morning", "AM", "in the morning"},
			LocalePreferred: map[string]string{"en-GB": "MANE", "en-US": "MANE"},
			Display:         map[string]string{"en-GB": "In the morning", "en-US": "In the morning"},
			Category:        CategoryTimeOfDay,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   24,
			DefaultTimes:    []string{"08:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			Latin:           "mane",
			FhirCode:        "AM",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       16,
		},
		{
			Code:            "NOCTE",
			Aliases:         []string{"nocte", "ON", "at night", "at bedtime", "HS"},
			LocalePreferred: map[string]string{"en-GB": "NOCTE", "en-US": "NOCTE"},
			Display:         map[string]string{"en-GB": "At night", "en-US": "At night"},
			Category:        CategoryTimeOfDay,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   24,
			DefaultTimes:    []string{"22:00"},
			MealRelation:    MealNone,
			Latin:           "nocte",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       17,
		},
		{
			Code:            "MIDI",
			Aliases:         []string{"midday", "noon", "lunchtime"},
			LocalePreferred: map[string]string{"en-GB": "MIDI", "en-US": "MIDI"},
			Display:         map[string]string{"en-GB": "At midday", "en-US": "At midday"},
			Category:        CategoryTimeOfDay,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   24,
			DefaultTimes:    []string{"12:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       18,
		},
		{
			Code:            "AM_PM",
			Aliases:         []string{"morning and evening", "AM+PM"},
			LocalePreferred: map[string]string{"en-GB": "AM+PM", "en-US": "AM+PM"},
			Display:         map[string]string{"en-GB": "Morning and evening", "en-US": "Morning and evening"},
			Category:        CategoryTimeOfDay,
			Frequency:       2,
			Period:          1,
			PeriodUnit:      PeriodDay,
			IntervalHours:   12,
			DefaultTimes:    []string{"08:00", "20:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       19,
		},
		// Meal-relative frequencies
		{
			Code:            "AC",
			Aliases:         []string{"a.c.", "before food", "before meals", "ante cibum"},
			LocalePreferred: map[string]string{"en-GB": "AC", "en-US": "AC"},
			Display:         map[string]string{"en-GB": "Before meals", "en-US": "Before meals"},
			Category:        CategoryMealRelative,
			Frequency:       0,
			Period:          0,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			WakingOnly:      true,
			MealRelation:    MealBefore,
			Latin:           "ante cibum",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       20,
		},
		{
			Code:            "PC",
			Aliases:         []string{"p.c.", "after food", "after meals", "post cibum"},
			LocalePreferred: map[string]string{"en-GB": "PC", "en-US": "PC"},
			Display:         map[string]string{"en-GB": "After meals", "en-US": "After meals"},
			Category:        CategoryMealRelative,
			Frequency:       0,
			Period:          0,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			WakingOnly:      true,
			MealRelation:    MealAfter,
			Latin:           "post cibum",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       21,
		},
		{
			Code:            "CC",
			Aliases:         []string{"c.c.", "with food", "with meals", "cum cibo"},
			LocalePreferred: map[string]string{"en-GB": "CC", "en-US": "CC"},
			Display:         map[string]string{"en-GB": "With meals", "en-US": "With meals"},
			Category:        CategoryMealRelative,
			Frequency:       0,
			Period:          0,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			WakingOnly:      true,
			MealRelation:    MealWith,
			Latin:           "cum cibo",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       22,
		},
		{
			Code:            "AC_HS",
			Aliases:         []string{"before meals and at bedtime", "AC+HS"},
			LocalePreferred: map[string]string{"en-GB": "AC+HS", "en-US": "AC+HS"},
			Display:         map[string]string{"en-GB": "Before meals and at bedtime", "en-US": "Before meals and at bedtime"},
			Category:        CategoryMealRelative,
			Frequency:       4,
			Period:          1,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{"07:30", "11:30", "17:30", "22:00"},
			MealRelation:    MealBefore,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       23,
		},
		// PRN (as-needed) frequencies
		{
			Code:            "PRN",
			Aliases:         []string{"p.r.n.", "as needed", "as required", "when required", "pro re nata"},
			LocalePreferred: map[string]string{"en-GB": "PRN", "en-US": "PRN"},
			Display:         map[string]string{"en-GB": "As needed", "en-US": "As needed"},
			Category:        CategoryPRN,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			AsNeeded:        true,
			MealRelation:    MealNone,
			Latin:           "pro re nata",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       24,
		},
		{
			Code:            "PRN_Q4H",
			Aliases:         []string{"PRN every 4 hours", "as needed max every 4 hours", "p.r.n. q4h"},
			LocalePreferred: map[string]string{"en-GB": "PRN Q4H", "en-US": "PRN Q4H"},
			Display:         map[string]string{"en-GB": "As needed, max every 4 hours", "en-US": "As needed, max every 4 hours"},
			Category:        CategoryPRN,
			Frequency:       1,
			Period:          4,
			PeriodUnit:      PeriodHour,
			IntervalHours:   4,
			DefaultTimes:    []string{},
			AsNeeded:        true,
			MaxPerDay:       6,
			MinInterval:     floatPtr(4),
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       25,
		},
		{
			Code:            "PRN_Q6H",
			Aliases:         []string{"PRN every 6 hours", "as needed max every 6 hours", "p.r.n. q6h"},
			LocalePreferred: map[string]string{"en-GB": "PRN Q6H", "en-US": "PRN Q6H"},
			Display:         map[string]string{"en-GB": "As needed, max every 6 hours", "en-US": "As needed, max every 6 hours"},
			Category:        CategoryPRN,
			Frequency:       1,
			Period:          6,
			PeriodUnit:      PeriodHour,
			IntervalHours:   6,
			DefaultTimes:    []string{},
			AsNeeded:        true,
			MaxPerDay:       4,
			MinInterval:     floatPtr(6),
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       26,
		},
		{
			Code:            "SOS",
			Aliases:         []string{"s.o.s.", "if needed", "si opus sit"},
			LocalePreferred: map[string]string{"en-GB": "SOS", "en-US": "SOS"},
			Display:         map[string]string{"en-GB": "If needed (single use)", "en-US": "If needed (single use)"},
			Category:        CategoryPRN,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			AsNeeded:        true,
			MaxPerDay:       1,
			MealRelation:    MealNone,
			Latin:           "si opus sit",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       27,
		},
		// One-off frequencies
		{
			Code:            "STAT",
			Aliases:         []string{"stat", "immediately", "now", "statim"},
			LocalePreferred: map[string]string{"en-GB": "STAT", "en-US": "STAT"},
			Display:         map[string]string{"en-GB": "Immediately", "en-US": "Immediately"},
			Category:        CategoryOneOff,
			Frequency:       1,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			MaxPerDay:       1,
			MealRelation:    MealNone,
			Latin:           "statim",
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       28,
		},
		{
			Code:            "ONCE",
			Aliases:         []string{"single dose", "one-off", "×1", "x1"},
			LocalePreferred: map[string]string{"en-GB": "ONCE", "en-US": "ONCE"},
			Display:         map[string]string{"en-GB": "Single dose", "en-US": "Single dose"},
			Category:        CategoryOneOff,
			Frequency:       1,
			PeriodUnit:      PeriodDay,
			DefaultTimes:    []string{},
			MaxPerDay:       1,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       29,
		},
		// Extended interval frequencies
		{
			Code:            "QOD",
			Aliases:         []string{"every other day", "alternate days", "on alternate days", "qod"},
			LocalePreferred: map[string]string{"en-GB": "QOD", "en-US": "QOD"},
			Display:         map[string]string{"en-GB": "Every other day", "en-US": "Every other day"},
			Category:        CategoryExtended,
			Frequency:       1,
			Period:          2,
			PeriodUnit:      PeriodDay,
			IntervalHours:   48,
			DefaultTimes:    []string{"08:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "QOD",
			FhirSystem:      fhirGTSSystem,
			SortOrder:       30,
		},
		{
			Code:            "WEEKLY",
			Aliases:         []string{"once a week", "weekly", "every week", "Q1W"},
			LocalePreferred: map[string]string{"en-GB": "WEEKLY", "en-US": "WEEKLY"},
			Display:         map[string]string{"en-GB": "Once weekly", "en-US": "Once weekly"},
			Category:        CategoryExtended,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodWeek,
			IntervalHours:   168,
			DefaultTimes:    []string{"08:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       31,
		},
		{
			Code:            "BIWEEKLY",
			Aliases:         []string{"every 2 weeks", "fortnightly", "Q2W"},
			LocalePreferred: map[string]string{"en-GB": "BIWEEKLY", "en-US": "BIWEEKLY"},
			Display:         map[string]string{"en-GB": "Every 2 weeks", "en-US": "Every 2 weeks"},
			Category:        CategoryExtended,
			Frequency:       1,
			Period:          2,
			PeriodUnit:      PeriodWeek,
			IntervalHours:   336,
			DefaultTimes:    []string{"08:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       32,
		},
		{
			Code:            "MONTHLY",
			Aliases:         []string{"once a month", "monthly", "Q1M"},
			LocalePreferred: map[string]string{"en-GB": "MONTHLY", "en-US": "MONTHLY"},
			Display:         map[string]string{"en-GB": "Once monthly", "en-US": "Once monthly"},
			Category:        CategoryExtended,
			Frequency:       1,
			Period:          1,
			PeriodUnit:      PeriodMonth,
			IntervalHours:   720,
			DefaultTimes:    []string{"08:00"},
			WakingOnly:      true,
			MealRelation:    MealNone,
			FhirCode:        "",
			FhirSystem:      "",
			SortOrder:       33,
		},
	}
}
