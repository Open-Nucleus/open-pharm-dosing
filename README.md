# open-pharma-dosing

A zero-dependency library for structured medication dosing frequency encoding, parsing, and FHIR R4 Timing conversion.

Maps clinical shorthand (BD, TDS, PRN, STAT, etc.) to structured timing data and FHIR resources, with locale support for UK and US conventions.

**Author:** Akanimoh Osutuk — [FibrinLab](https://github.com/FibrinLab)
**Licence:** Apache 2.0

---

## The Problem

Every prescribing system deals with dosing frequencies — BD, TDS, OD, QDS, PRN, STAT, nocte, mane — yet there is no open-source, standalone library that maps these to structured timing data with bidirectional FHIR conversion.

The UK says BD, the US says BID. Both mean "twice daily, every 12 hours, typically at 08:00 and 20:00." Clinicians think in shorthand. FHIR thinks in `{ "frequency": 2, "period": 1, "periodUnit": "d" }`. This library bridges the gap.

## Standards & Provenance

Every code in this library is traceable to an authoritative source. No codes are invented.

| Source | What it provides | Reference |
|--------|-----------------|-----------|
| **HL7 FHIR v3-GTSAbbreviation** | Formal coded timing abbreviations (QD, BID, TID, QID, Q1H–Q8H, QOD, AM, BED, WK, MO) | [terminology.hl7.org](https://terminology.hl7.org/6.5.0/CodeSystem-v3-GTSAbbreviation.html) |
| **FHIR R4 EventTiming** | `Timing.repeat.when` codes (MORN, NIGHT, AC, PC, HS, C) | [hl7.org/fhir/R4](https://hl7.org/fhir/R4/valueset-event-timing.html) |
| **NHS Dose Syntax Implementation Guide** | UK guidance for populating FHIR Dosage structures | [nhsconnect.github.io](https://nhsconnect.github.io/Dose-Syntax-Implementation/) |
| **NHS Dose Syntax API Standards** | National API standard for dose syntax in prescribing systems | [digital.nhs.uk](https://digital.nhs.uk/developer/api-catalogue/dose-syntax-standards) |
| **NHS App Medical Records** | Official abbreviations list (b.d., t.d.s., q.d.s., o.d., p.r.n., nocte, stat, a.c., p.c.) | [nhs.uk](https://www.nhs.uk/nhs-app/help/health-records-in-the-nhs-app/abbreviations-commonly-found-in-medical-records/) |
| **NHS Scotland Dose Syntax Recommendations** | Structured dose instruction standard for Scottish prescribing (2015) | [scimp.scot.nhs.uk](https://www.scimp.scot.nhs.uk/nhs-dose-syntax-recommendations-2015) |

Each frequency code is classified by source tier:

- **S1 — HL7 GTS:** Code exists in the HL7 v3-GTSAbbreviation CodeSystem (formal international standard)
- **S2 — NHS Convention:** Documented in NHS prescribing guidance or the NHS App abbreviations list
- **S3 — Clinical Extension:** Common prescribing pattern not in S1/S2; sourced from BNF usage and Latin tradition

## Frequency Registry

33 dosing frequency codes across 7 categories:

| Code | Display | Category | Source | FHIR Code |
|------|---------|----------|--------|-----------|
| OD | Once daily | Regular | S1+S2 | QD |
| BD | Twice daily | Regular | S1+S2 | BID |
| TDS | Three times daily | Regular | S1+S2 | TID |
| QDS | Four times daily | Regular | S1+S2 | QID |
| 5X_DAILY | Five times daily | Regular | S3 | — |
| Q1H | Every 1 hour | Interval | S1+S2 | Q1H |
| Q2H | Every 2 hours | Interval | S1+S2 | — |
| Q4H | Every 4 hours | Interval | S1+S2 | Q4H |
| Q6H | Every 6 hours | Interval | S1+S2 | Q6H |
| Q8H | Every 8 hours | Interval | S1+S2 | — |
| Q12H | Every 12 hours | Interval | S3 | — |
| Q24H | Every 24 hours | Interval | S3 | — |
| Q36H | Every 36 hours | Interval | S3 | — |
| Q48H | Every 48 hours | Interval | S3 | — |
| Q72H | Every 72 hours | Interval | S3 | — |
| MANE | In the morning | Time of Day | S1+S2 | AM |
| NOCTE | At night | Time of Day | S1+S2 | — |
| MIDI | At midday | Time of Day | S3 | — |
| AM_PM | Morning and evening | Time of Day | S3 | — |
| AC | Before meals | Meal Relative | S2 | — |
| PC | After meals | Meal Relative | S2 | — |
| CC | With meals | Meal Relative | S3 | — |
| AC_HS | Before meals and at bedtime | Meal Relative | S3 | — |
| PRN | As needed | PRN | S2 | — |
| PRN_Q4H | As needed, max every 4 hours | PRN | S3 | — |
| PRN_Q6H | As needed, max every 6 hours | PRN | S3 | — |
| SOS | If needed (single use) | PRN | S3 | — |
| STAT | Immediately | One-Off | S2 | — |
| ONCE | Single dose | One-Off | S3 | — |
| QOD | Every other day | Extended | S1+S2 | QOD |
| WEEKLY | Once weekly | Extended | S1 | — |
| BIWEEKLY | Every 2 weeks | Extended | S3 | — |
| MONTHLY | Once monthly | Extended | S1 | — |

## Installation

### Go (canonical implementation)

```bash
go get github.com/FibrinLab/open-pharma-dosing
```

### Dart (coming soon)

```bash
dart pub add open_pharma_dosing
```

### Python (coming soon)

```bash
pip install open-pharma-dosing
```

## Usage

### Parse clinical shorthand

```go
import dosing "github.com/FibrinLab/open-pharma-dosing"

// Parse accepts canonical codes, aliases, mixed case, and punctuation variants
fc, _ := dosing.Parse("BD")           // → FrequencyCode{Code: "BD", ...}
fc, _ = dosing.Parse("b.i.d.")        // → FrequencyCode{Code: "BD", ...}
fc, _ = dosing.Parse("twice daily")   // → FrequencyCode{Code: "BD", ...}
fc, _ = dosing.Parse("every 4 hours") // → FrequencyCode{Code: "Q4H", ...}
fc, _ = dosing.Parse("TID")           // → FrequencyCode{Code: "TDS", ...}
fc, _ = dosing.Parse("p.r.n.")        // → FrequencyCode{Code: "PRN", ...}
```

### Look up frequency codes

```go
// Get by canonical code
fc, err := dosing.Get("TDS")
fmt.Println(fc.Frequency)       // 3
fmt.Println(fc.IntervalHours)   // 8
fmt.Println(fc.DefaultTimes)    // ["08:00", "14:00", "20:00"]
fmt.Println(fc.Latin)           // "ter die sumendus"

// List all codes (sorted by SortOrder)
all := dosing.List()

// Filter by category
prn := dosing.List(dosing.WithCategory(dosing.CategoryPRN))

// Search by partial match on code, alias, or display text
results := dosing.Search("daily")   // matches OD, BD, TDS, QDS, 5X_DAILY
results = dosing.Search("morning")  // matches MANE, AM_PM
```

### Convert to FHIR R4

```go
// Frequency code → FHIR Timing JSON
timingJSON, _ := dosing.ToFhirTiming("BD")
// Output:
// {
//   "repeat": {
//     "frequency": 2,
//     "period": 1,
//     "periodUnit": "d",
//     "timeOfDay": ["08:00", "20:00"]
//   },
//   "code": {
//     "coding": [{
//       "system": "http://terminology.hl7.org/CodeSystem/v3-GTSAbbreviation",
//       "code": "BID"
//     }]
//   }
// }

// FHIR Timing JSON → frequency code
fc, _ := dosing.FromFhirTiming(timingJSON)
fmt.Println(fc.Code) // "BD"
```

### Convert full dosing instructions

```go
// Build a dosing instruction
instruction := &dosing.DosingInstruction{
    Frequency: fc,
    Dose:      &dosing.Dose{Value: 500, Unit: "mg"},
    Route:     "PO",
}

// Convert to FHIR Dosage
dosageJSON, _ := dosing.ToFhirDosage(instruction)

// Convert back
result, _ := dosing.FromFhirDosage(dosageJSON)
fmt.Println(result.Frequency.Code) // "BD"
fmt.Println(result.Dose.Value)     // 500
fmt.Println(result.Route)          // "PO"
```

## FHIR Roundtrip Fidelity

A key invariant of this library:

```
FromFhirTiming(ToFhirTiming(code)) == code
```

This holds for all 33 supported frequency codes. Three codes have documented structural ambiguities where the roundtrip resolves to an equivalent code:

| Input | Roundtrip Result | Reason |
|-------|-----------------|--------|
| ONCE | STAT | Both produce `count: 1`; STAT is the default |
| AM_PM | BD | Both produce `frequency: 2, period: 1/d, times: [08:00, 20:00]` |
| SOS | PRN | Both produce `asNeeded: true` without period constraints |

These are structurally indistinguishable in FHIR and resolve to the more common code.

## Repository Structure

```
open-pharma-dosing/
├── README.md
├── LICENSE
├── spec.md                         # Full specification
├── CLAUDE.md                       # AI assistant instructions
├── data/
│   └── frequencies.json            # Canonical registry (reference for ports)
├── go/                             # Go implementation (canonical)
│   ├── go.mod
│   ├── dosing.go                   # Types, constants, embedded registry
│   ├── parse.go                    # Parser
│   ├── fhir.go                     # FHIR R4 converter
│   ├── dosing_test.go
│   ├── parse_test.go
│   └── fhir_test.go
├── dart/                           # Dart/Flutter port (planned)
├── python/                         # Python port (planned)
└── docs/                           # Documentation (planned)
```

## Design Decisions

**Zero dependencies.** The library has no external dependencies in any language implementation. The frequency registry is embedded as compiled data, not loaded from files at runtime.

**NHS-first, internationally compatible.** Canonical codes use UK convention (BD, TDS, QDS, OD) with US aliases (BID, TID, QID, QD). The parser accepts both. The FHIR converter uses the HL7 GTS codes (BID, TID, QID) which are the international standard.

**Registry as code, not config.** The 33 frequency entries are hand-written Go struct literals, type-checked at compile time. The `data/frequencies.json` file is an export for documentation and for the Dart/Python ports — the Go implementation never reads it.

**FHIR types are internal.** The library uses internal Go structs for FHIR Timing/Dosage construction but exposes `[]byte` (JSON) at the public API boundary. This keeps the API simple and avoids depending on any FHIR library.

## Development

```bash
cd go && go build ./...        # Build
cd go && go vet ./...          # Lint
cd go && go test ./...         # Run all tests
cd go && go test -v -run TestFhirRoundtripAllCodes  # FHIR roundtrip invariant
```

## Roadmap

- [x] **Phase 1:** Core Go library — types, registry, parser, FHIR converter
- [ ] **Phase 2:** Text generation, schedule generator, validation
- [ ] **Phase 3:** Dart and Python ports
- [ ] **Phase 4:** Locale support (en-GB, en-US, fr, es, sw, ha, yo), ParseInstruction
- [ ] **Phase 5:** Complex regimens (tapering, split-dose, cyclical), EPMA integration

## Contributing

Contributions are welcome, particularly:

- **Clinical review** — are the default administration times sensible?
- **Locale translations** — especially African languages (Swahili, Hausa, Yoruba)
- **Missing frequency codes** — if your prescribing system uses codes not in the registry
- **FHIR edge cases** — unusual Timing structures that should map to a code

## References

1. HL7 FHIR v3-GTSAbbreviation CodeSystem — [terminology.hl7.org/CodeSystem/v3-GTSAbbreviation](https://terminology.hl7.org/6.5.0/CodeSystem-v3-GTSAbbreviation.html)
2. FHIR R4 TimingAbbreviation ValueSet — [hl7.org/fhir/R4/valueset-timing-abbreviation.html](https://hl7.org/fhir/R4/valueset-timing-abbreviation.html)
3. FHIR R4 EventTiming ValueSet — [hl7.org/fhir/R4/valueset-event-timing.html](https://hl7.org/fhir/R4/valueset-event-timing.html)
4. NHS Dose Syntax Implementation Guide — [nhsconnect.github.io/Dose-Syntax-Implementation](https://nhsconnect.github.io/Dose-Syntax-Implementation/)
5. NHS Dose Syntax API Standards — [digital.nhs.uk/developer/api-catalogue/dose-syntax-standards](https://digital.nhs.uk/developer/api-catalogue/dose-syntax-standards)
6. NHS Scotland Dose Syntax Recommendations (2015) — [scimp.scot.nhs.uk](https://www.scimp.scot.nhs.uk/nhs-dose-syntax-recommendations-2015)
7. NHS App Medical Records Abbreviations — [nhs.uk/nhs-app/help/health-records](https://www.nhs.uk/nhs-app/help/health-records-in-the-nhs-app/abbreviations-commonly-found-in-medical-records/)
8. Community Pharmacy England — Dose Syntax Interoperability — [psnc.org.uk](https://psnc.org.uk/contract-it/pharmacy-it/standards-and-interoperability-it/standard-dosage-syntax-interoperability)

---

*open-pharma-dosing — FibrinLab*
*Because "BD" shouldn't need 47 lines of FHIR XML.*
