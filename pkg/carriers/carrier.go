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

// CarrierWithAliases is an optional interface for carriers providing known alternate identifiers.
type CarrierWithAliases interface {
	Carrier
	Aliases() []string
}

// Registry manages the set of available carrier adapters.
type Registry struct {
	mu       sync.RWMutex
	carriers map[string]Carrier
	aliases  map[string]Carrier
}

var (
	defaultRegistry *Registry
	once            sync.Once
)

// GetRegistry returns the singleton carrier registry.
func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = &Registry{
			carriers: make(map[string]Carrier),
			aliases:  make(map[string]Carrier),
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

	// Netherlands carriers & aliases
	postnl := NewPostNLCarrier()
	r.Register(postnl)
	r.RegisterAlias("postnl-international", postnl)
	r.RegisterAlias("post-nl", postnl)
	r.RegisterAlias("postnl-nl", postnl)

	// Belgium carriers & aliases
	bpost := NewBpostCarrier()
	r.Register(bpost)
	r.RegisterAlias("bpost-local", bpost)
	r.RegisterAlias("bpost-be", bpost)
	r.RegisterAlias("belgian-post", bpost)

	// Luxembourg carrier & aliases
	luxPost := NewLuxembourgPostCarrier()
	r.Register(luxPost)
	r.RegisterAlias("post-luxembourg", luxPost)
	r.RegisterAlias("post-lu", luxPost)
}

// Register adds a carrier to the registry and registers its aliases if implemented.
func (r *Registry) Register(c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carriers[strings.ToLower(c.Slug())] = c
	if ca, ok := c.(CarrierWithAliases); ok {
		if r.aliases == nil {
			r.aliases = make(map[string]Carrier)
		}
		for _, alias := range ca.Aliases() {
			aliasKey := strings.ToLower(alias)
			// Don't overwrite an existing primary carrier slug
			if _, exists := r.carriers[aliasKey]; !exists {
				r.aliases[aliasKey] = c
			}
		}
	}
}

// RegisterAlias registers an alternative slug/alias that maps to a carrier.
func (r *Registry) RegisterAlias(alias string, c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.aliases == nil {
		r.aliases = make(map[string]Carrier)
	}
	r.aliases[strings.ToLower(alias)] = c
}

// Get finds a carrier by its primary slug or known alias.
func (r *Registry) Get(slug string) (Carrier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := strings.ToLower(slug)
	c, ok := r.carriers[key]
	if ok {
		return c, true
	}
	if r.aliases != nil {
		c, ok = r.aliases[key]
		if ok {
			return c, true
		}
	}
	return nil, false
}

// GetAlias finds a carrier specifically by registered alias.
func (r *Registry) GetAlias(alias string) (Carrier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.aliases == nil {
		return nil, false
	}
	c, ok := r.aliases[strings.ToLower(alias)]
	return c, ok
}

// Aliases returns a copy of all registered aliases and their corresponding carriers.
func (r *Registry) Aliases() map[string]Carrier {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make(map[string]Carrier, len(r.aliases))
	for k, v := range r.aliases {
		res[k] = v
	}
	return res
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
		if seen[info.Slug] {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(info.Slug), q) ||
			strings.Contains(strings.ToLower(info.Name), q) ||
			strings.Contains(strings.ToLower(info.Country), q) ||
			strings.Contains(strings.ToLower(info.CountryName), q) {
			seen[info.Slug] = true
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
