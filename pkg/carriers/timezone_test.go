package carriers

import (
	"testing"
	"time"
)

func TestTimezoneResolveLocation(t *testing.T) {
	cases := []struct {
		input       string
		expectedLoc string
	}{
		{"Asia/Shanghai", "Asia/Shanghai"},
		{"Europe/Madrid", "Europe/Madrid"},
		{"CN", "Asia/Shanghai"},
		{"China", "Asia/Shanghai"},
		{"ES", "Europe/Madrid"},
		{"Spain", "Europe/Madrid"},
		{"España", "Europe/Madrid"},
		{"IT", "Europe/Rome"},
		{"PT", "Europe/Lisbon"},
		{"cainiao", "Asia/Shanghai"},
		{"correosexpress", "Europe/Madrid"},
		{"correos-express", "Europe/Madrid"},
		{"inpost", "Europe/Madrid"},
		{"CST", "Asia/Shanghai"},
		{"CET", "Europe/Madrid"},
		{"CEST", "Europe/Madrid"},
		{"UTC", "UTC"},
		{"", "UTC"},
		{"unknown-country", "UTC"},
	}

	for _, tc := range cases {
		loc := ResolveLocation(tc.input)
		if loc.String() != tc.expectedLoc {
			t.Errorf("ResolveLocation(%q) = %q, expected %q", tc.input, loc.String(), tc.expectedLoc)
		}
	}
}

func TestParseLocalTimeWithZone(t *testing.T) {
	// 1. Without explicit offset: Asia/Shanghai (UTC+8)
	// 10:00:00 CST -> 02:00:00 UTC
	tShanghai, err := ParseLocalTimeWithZone("2006-01-02 15:04:05", "2024-03-15 10:00:00", "Asia/Shanghai")
	if err != nil {
		t.Fatalf("unexpected error parsing Shanghai time: %v", err)
	}
	expectedShanghaiUTC := time.Date(2024, 3, 15, 2, 0, 0, 0, time.UTC)
	if !tShanghai.Equal(expectedShanghaiUTC) {
		t.Errorf("expected Shanghai time %v UTC, got %v", expectedShanghaiUTC, tShanghai)
	}

	// 2. Without explicit offset: Europe/Madrid in winter (CET, UTC+1)
	// 09:00:00 CET -> 08:00:00 UTC
	tMadridWinter, err := ParseLocalTimeWithZone("2006-01-02 15:04:05", "2024-03-15 09:00:00", "Europe/Madrid")
	if err != nil {
		t.Fatalf("unexpected error parsing Madrid winter time: %v", err)
	}
	expectedMadridWinterUTC := time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC)
	if !tMadridWinter.Equal(expectedMadridWinterUTC) {
		t.Errorf("expected Madrid winter time %v UTC, got %v", expectedMadridWinterUTC, tMadridWinter)
	}

	// 3. Chronological assertion:
	// Event in Madrid at 09:00:00 CET (08:00:00 UTC) must be AFTER event in Shanghai at 10:00:00 CST (02:00:00 UTC)
	if !tMadridWinter.After(tShanghai) {
		t.Errorf("Madrid event at 09:00 CET (%v) should be chronologically AFTER Shanghai event at 10:00 CST (%v)",
			tMadridWinter, tShanghai)
	}

	// 4. Without explicit offset: Europe/Madrid in summer (CEST, UTC+2)
	// 10:00:00 CEST -> 08:00:00 UTC
	tMadridSummer, err := ParseLocalTimeWithZone("2006-01-02 15:04:05", "2024-07-15 10:00:00", "Europe/Madrid")
	if err != nil {
		t.Fatalf("unexpected error parsing Madrid summer time: %v", err)
	}
	expectedMadridSummerUTC := time.Date(2024, 7, 15, 8, 0, 0, 0, time.UTC)
	if !tMadridSummer.Equal(expectedMadridSummerUTC) {
		t.Errorf("expected Madrid summer time %v UTC, got %v", expectedMadridSummerUTC, tMadridSummer)
	}

	// 5. With explicit offset: RFC3339 "+08:00"
	tExplicit, err := ParseLocalTimeWithZone(time.RFC3339, "2024-03-15T10:00:00+08:00", "Europe/Madrid")
	if err != nil {
		t.Fatalf("unexpected error parsing explicit RFC3339: %v", err)
	}
	if !tExplicit.Equal(expectedShanghaiUTC) {
		t.Errorf("expected explicit offset to yield %v, got %v", expectedShanghaiUTC, tExplicit)
	}
}

func TestNormalizeEventTime(t *testing.T) {
	// Given a wall-clock time 10:00:00 without offset
	rawWallClock := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)

	// Normalized to Asia/Shanghai (UTC+8) -> 02:00:00 UTC
	normShanghai := NormalizeEventTime(rawWallClock, "Asia/Shanghai")
	if normShanghai.Hour() != 2 || normShanghai.Day() != 15 {
		t.Errorf("expected 02:00:00 UTC for Shanghai, got %v", normShanghai)
	}

	// Normalized to Europe/Madrid (UTC+1 in March) -> 09:00:00 UTC
	normMadrid := NormalizeEventTime(rawWallClock, "Europe/Madrid")
	if normMadrid.Hour() != 9 || normMadrid.Day() != 15 {
		t.Errorf("expected 09:00:00 UTC for Madrid, got %v", normMadrid)
	}
}

func TestCarrierParsersTimezoneNormalization(t *testing.T) {
	// 1. Cainiao parser defaults to Asia/Shanghai (UTC+8) when no explicit offset
	det := cainiaoDetailItem{
		TimeStr: "2024-03-15 10:00:00",
		Desc:    "Departed sorting center",
	}
	cTime, raw := parseCainiaoTime(det)
	if raw != "2024-03-15 10:00:00" {
		t.Errorf("expected raw date preserved, got %q", raw)
	}
	expectedCainiaoUTC := time.Date(2024, 3, 15, 2, 0, 0, 0, time.UTC)
	if !cTime.Equal(expectedCainiaoUTC) {
		t.Errorf("expected Cainiao time %v UTC, got %v", expectedCainiaoUTC, cTime)
	}

	// 2. Cainiao parser respects explicit det.TimeZone if present (e.g. GMT+1)
	detMadrid := cainiaoDetailItem{
		TimeStr:  "2024-03-15 09:00:00",
		TimeZone: "GMT+1",
		Desc:     "Arrived at local facility",
	}
	cMadridTime, _ := parseCainiaoTime(detMadrid)
	expectedMadridUTC := time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC)
	if !cMadridTime.Equal(expectedMadridUTC) {
		t.Errorf("expected Cainiao with GMT+1 %v UTC, got %v", expectedMadridUTC, cMadridTime)
	}

	// 3. Correos Express parser uses Europe/Madrid (CET in winter, CEST in summer)
	ceWinter := parseCorreosExpressDate("15/03/2024", "09:00:00")
	if !ceWinter.Equal(expectedMadridUTC) {
		t.Errorf("expected Correos Express winter time %v UTC, got %v", expectedMadridUTC, ceWinter)
	}

	ceSummer := parseCorreosExpressDate("15/07/2024", "10:00:00")
	expectedSummerUTC := time.Date(2024, 7, 15, 8, 0, 0, 0, time.UTC)
	if !ceSummer.Equal(expectedSummerUTC) {
		t.Errorf("expected Correos Express summer time %v UTC, got %v", expectedSummerUTC, ceSummer)
	}

	// 4. Verify Madrid 09:00 CET is after Cainiao 10:00 CST
	if !ceWinter.After(cTime) {
		t.Errorf("expected Correos Express 09:00 CET (%v) to be after Cainiao 10:00 CST (%v)", ceWinter, cTime)
	}
}
