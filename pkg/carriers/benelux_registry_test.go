// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"strings"
	"testing"
)

func TestBeneluxCarriersRegisteredInDefaultRegistry(t *testing.T) {
	reg := GetRegistry()

	expectedSlugsAndAliases := []struct {
		key          string
		expectedSlug string
		country      string
	}{
		{"postnl", "postnl", "NL"},
		{"postnl-international", "postnl", "NL"},
		{"bpost", "bpost", "BE"},
		{"bpost-local", "bpost", "BE"},
		{"luxembourg-post", "luxembourg-post", "LU"},
		{"post-luxembourg", "luxembourg-post", "LU"},
	}

	for _, tc := range expectedSlugsAndAliases {
		t.Run("Get_"+tc.key, func(t *testing.T) {
			carrier, ok := reg.Get(tc.key)
			if !ok || carrier == nil {
				t.Fatalf("expected Get(%q) to find a carrier in registry, but got ok=%v, carrier=%v", tc.key, ok, carrier)
			}
			if carrier.Country() != tc.country {
				t.Errorf("expected carrier for %q to have country %q, got %q", tc.key, tc.country, carrier.Country())
			}
			if tc.key == tc.expectedSlug && carrier.Slug() != tc.expectedSlug {
				t.Errorf("expected primary carrier slug %q, got %q", tc.expectedSlug, carrier.Slug())
			}
		})

		t.Run("Get_CaseInsensitive_"+strings.ToUpper(tc.key), func(t *testing.T) {
			upperKey := strings.ToUpper(tc.key)
			carrier, ok := reg.Get(upperKey)
			if !ok || carrier == nil {
				t.Fatalf("expected Get(%q) case-insensitive to find carrier", upperKey)
			}
			if carrier.Country() != tc.country {
				t.Errorf("expected carrier for %q to have country %q, got %q", upperKey, tc.country, carrier.Country())
			}
		})
	}
}

func TestBeneluxCarriersInGlobalList(t *testing.T) {
	reg := GetRegistry()
	allCarriers := reg.List("")

	foundPostNL := false
	foundBpost := false
	foundLuxPost := false

	for _, c := range allCarriers {
		switch c.Slug {
		case "postnl":
			foundPostNL = true
			if c.Country != "NL" || c.CountryName != "Netherlands" {
				t.Errorf("unexpected PostNL info in global list: %+v", c)
			}
		case "bpost":
			foundBpost = true
			if c.Country != "BE" || c.CountryName != "Belgium" {
				t.Errorf("unexpected Bpost info in global list: %+v", c)
			}
		case "luxembourg-post":
			foundLuxPost = true
			if c.Country != "LU" || c.CountryName != "Luxembourg" {
				t.Errorf("unexpected POST Luxembourg info in global list: %+v", c)
			}
		}
	}

	if !foundPostNL {
		t.Errorf("expected 'postnl' to be present in global GetRegistry().List('')")
	}
	if !foundBpost {
		t.Errorf("expected 'bpost' to be present in global GetRegistry().List('')")
	}
	if !foundLuxPost {
		t.Errorf("expected 'luxembourg-post' to be present in global GetRegistry().List('')")
	}
}

func TestBeneluxCarriersSpecificQueries(t *testing.T) {
	reg := GetRegistry()

	queries := []struct {
		query        string
		expectedSlug string
	}{
		{"postnl", "postnl"},
		{"netherlands", "postnl"},
		{"bpost", "bpost"},
		{"belgium", "bpost"},
		{"luxembourg", "luxembourg-post"},
	}

	for _, tc := range queries {
		t.Run("List_"+tc.query, func(t *testing.T) {
			results := reg.List(tc.query)
			found := false
			for _, item := range results {
				if item.Slug == tc.expectedSlug {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected List(%q) to include carrier with slug %q", tc.query, tc.expectedSlug)
			}
		})
	}
}

func TestBeneluxAliasesRegistered(t *testing.T) {
	reg := GetRegistry()

	expectedAliases := []string{
		"postnl-international",
		"post-nl",
		"postnl-nl",
		"bpost-local",
		"bpost-be",
		"post-luxembourg",
		"post-lu",
	}

	for _, alias := range expectedAliases {
		t.Run("GetAlias_"+alias, func(t *testing.T) {
			carrier, ok := reg.GetAlias(alias)
			if !ok || carrier == nil {
				t.Errorf("expected GetAlias(%q) to return true and non-nil carrier", alias)
			}
		})
	}

	allAliases := reg.Aliases()
	for _, alias := range expectedAliases {
		if _, ok := allAliases[alias]; !ok {
			t.Errorf("expected reg.Aliases() to contain key %q", alias)
		}
	}

	// Unknown carrier lookup returns false
	if _, ok := reg.Get("unknown-carrier-xyz"); ok {
		t.Errorf("expected Get for non-existent carrier to return false")
	}

	// BpostLocal carrier aliases coverage
	bLocal := NewBpostLocalCarrier()
	if len(bLocal.Aliases()) == 0 {
		t.Errorf("expected bpost-local carrier to have aliases")
	}

	// Registry with nil aliases Get coverage
	emptyReg := &Registry{carriers: make(map[string]Carrier)}
	if _, ok := emptyReg.Get("nonexistent"); ok {
		t.Errorf("expected Get on registry with nil aliases to return false")
	}
}
