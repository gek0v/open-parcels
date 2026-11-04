// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"open-parcels/pkg/models"
)

// Carrier is the common interface implemented by all carrier adapters.
type Carrier interface {
	Slug() string
	Name() string
	Country() string
	CountryName() string
	Type() models.CarrierType
	Website() string
	TrackingURL(trackingNumber string) string
	Info() models.CarrierInfo
	Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error)
}

// Registry manages the set of available carrier adapters and alias mappings.
type Registry struct {
	mu             sync.RWMutex
	carriers       map[string]Carrier
	aliases        map[string]string
	reverseAliases map[string][]string
}

var (
	defaultRegistry *Registry
	once            sync.Once
)

// GetRegistry returns the singleton carrier registry.
func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = &Registry{
			carriers:       make(map[string]Carrier),
			aliases:        make(map[string]string),
			reverseAliases: make(map[string][]string),
		}
		registerDefaultCarriers(defaultRegistry)
	})
	return defaultRegistry
}

func registerDefaultCarriers(r *Registry) {
	r.Register(NewCainiaoCarrier())
	r.Register(NewCorreosSpainCarrier())
	r.Register(NewCorreosExpressCarrier())
	r.Register(NewCTTExpressCarrier())
	r.Register(NewDHLExpressCarrier())
	r.Register(NewDHLParcelCarrier())
	r.Register(NewFedExCarrier())
	r.Register(NewFourPXCarrier())
	r.Register(NewGLSCarrier())
	r.Register(NewInPostCarrier())
	r.Register(NewMRWCarrier())
	r.Register(NewPaackCarrier())
	r.Register(NewSEURCarrier())
	r.Register(NewUPSCarrier())
	r.Register(NewUSPSCarrier())
	r.Register(NewYunExpressCarrier())
	r.Register(NewRoyalMailCarrier())
	r.Register(NewColissimoCarrier())
	r.Register(NewAirCargoCarrier())
	r.Register(NewOceanContainerCarrier())

	// Spanish carriers
	r.Register(NewNacexCarrier())
	r.Register(NewSendingCarrier())
	r.Register(NewRedurCarrier())
	r.Register(NewTipsaCarrier())
	r.Register(NewZelerisCarrier())
	r.Register(NewEnvialiaCarrier())
	r.Register(NewOntimeCarrier())
	r.Register(NewGLSSpainCarrier())

	// Aliases
	r.RegisterAlias("mrw-spain", "mrw")
	r.RegisterAlias("correosexpress", "correos-express")
	r.RegisterAlias("tourline", "ctt-express")
	r.RegisterAlias("nacex-spain", "nacex")
	r.RegisterAlias("sending-es", "sending")
	r.RegisterAlias("redur-spain", "redur")
	r.RegisterAlias("redur-spain-ref", "redur")
	r.RegisterAlias("redur-tracking", "redur")
	r.RegisterAlias("inpost-es", "inpost")
	r.RegisterAlias("asm-gls-spain", "gls-spain")
}

// Register adds a carrier to the registry.
func (r *Registry) Register(c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.carriers == nil {
		r.carriers = make(map[string]Carrier)
	}
	r.carriers[strings.ToLower(strings.TrimSpace(c.Slug()))] = c
}

// RegisterAlias registers an alternate slug for a canonical carrier slug.
func (r *Registry) RegisterAlias(alias, canonicalSlug string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	a := strings.ToLower(strings.TrimSpace(alias))
	c := strings.ToLower(strings.TrimSpace(canonicalSlug))
	if a == "" || c == "" || a == c {
		return
	}
	if r.aliases == nil {
		r.aliases = make(map[string]string)
	}
	if r.reverseAliases == nil {
		r.reverseAliases = make(map[string][]string)
	}
	r.aliases[a] = c

	found := false
	for _, rev := range r.reverseAliases[c] {
		if rev == a {
			found = true
			break
		}
	}
	if !found {
		r.reverseAliases[c] = append(r.reverseAliases[c], a)
	}
}

// Get finds a carrier by its slug or alias.
func (r *Registry) Get(slug string) (Carrier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	norm := strings.ToLower(strings.TrimSpace(slug))
	if norm == "" {
		return nil, false
	}

	if c, ok := r.carriers[norm]; ok {
		return c, true
	}

	// Follow alias forward chain
	visited := make(map[string]bool)
	curr := norm
	for {
		if visited[curr] {
			break
		}
		visited[curr] = true

		canonical, ok := r.aliases[curr]
		if !ok {
			break
		}
		if c, found := r.carriers[canonical]; found {
			return c, true
		}
		curr = canonical
	}

	// Check reverse aliases
	if revs, ok := r.reverseAliases[norm]; ok {
		for _, rev := range revs {
			if c, found := r.carriers[rev]; found {
				return c, true
			}
		}
	}

	return nil, false
}

// List returns metadata for all registered carriers.
func (r *Registry) List(query string) []models.CarrierInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	var list []models.CarrierInfo
	seen := make(map[string]bool)

	for _, c := range r.carriers {
		info := c.Info()
		canonicalSlug := strings.ToLower(strings.TrimSpace(info.Slug))
		if seen[canonicalSlug] {
			continue
		}

		matched := false
		if q == "" ||
			strings.Contains(canonicalSlug, q) ||
			strings.Contains(strings.ToLower(info.Name), q) ||
			strings.Contains(strings.ToLower(info.Country), q) ||
			strings.Contains(strings.ToLower(info.CountryName), q) {
			matched = true
		} else {
			// Check if any alias for this carrier matches the query
			for alias, target := range r.aliases {
				if (target == canonicalSlug || strings.EqualFold(target, info.Slug)) && strings.Contains(alias, q) {
					matched = true
					break
				}
			}
			if !matched {
				if revs, ok := r.reverseAliases[canonicalSlug]; ok {
					for _, rev := range revs {
						if strings.Contains(rev, q) {
							matched = true
							break
						}
					}
				}
			}
		}

		if matched {
			seen[canonicalSlug] = true
			list = append(list, info)
		}
	}
	return list
}

// AllCarriers returns all registered Carrier instances.
func (r *Registry) AllCarriers() map[string]Carrier {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make(map[string]Carrier, len(r.carriers))
	for k, v := range r.carriers {
		res[k] = v
	}
	return res
}

// BaseCarrier provides shared utility methods for carrier implementations.
type BaseCarrier struct {
	slug        string
	name        string
	country     string
	countryName string
	carrierType models.CarrierType
	website     string
	urlTemplate string
}

func NewBaseCarrier(slug, name, country, countryName string, cType models.CarrierType, website, urlTemplate string) BaseCarrier {
	return BaseCarrier{
		slug:        slug,
		name:        name,
		country:     country,
		countryName: countryName,
		carrierType: cType,
		website:     website,
		urlTemplate: urlTemplate,
	}
}

func (b BaseCarrier) Slug() string             { return b.slug }
func (b BaseCarrier) Name() string             { return b.name }
func (b BaseCarrier) Country() string          { return b.country }
func (b BaseCarrier) CountryName() string      { return b.countryName }
func (b BaseCarrier) Type() models.CarrierType { return b.carrierType }
func (b BaseCarrier) Website() string          { return b.website }
func (b BaseCarrier) TrackingURL(num string) string {
	if b.urlTemplate == "" {
		return ""
	}
	return fmt.Sprintf(b.urlTemplate, num)
}

func (b BaseCarrier) Info() models.CarrierInfo {
	return models.CarrierInfo{
		Slug:        b.slug,
		Name:        b.name,
		Country:     b.country,
		CountryName: b.countryName,
		Type:        b.carrierType,
		Website:     b.website,
		TrackingURL: b.urlTemplate,
	}
}
