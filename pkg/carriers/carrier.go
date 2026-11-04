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

// Registry manages the set of available carrier adapters.
type Registry struct {
	mu       sync.RWMutex
	carriers map[string]Carrier
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
	ptPost := NewPortugalPostCarrier()
	r.Register(ptPost)
	r.RegisterAlias("ctt-portugal", ptPost)
	r.RegisterAlias("ctt", ptPost)
	r.Register(NewDPDPortugalCarrier())
	r.Register(NewVaspExpressoCarrier())
	r.Register(NewDelnextCarrier())
}

// Register adds a carrier to the registry.
func (r *Registry) Register(c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carriers[strings.ToLower(c.Slug())] = c
}

// RegisterAlias registers an existing carrier under an alias slug.
func (r *Registry) RegisterAlias(alias string, c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carriers[strings.ToLower(strings.TrimSpace(alias))] = c
}

// Get finds a carrier by its slug.
func (r *Registry) Get(slug string) (Carrier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.carriers[strings.ToLower(slug)]
	return c, ok
}

// List returns metadata for all registered carriers.
func (r *Registry) List(query string) []models.CarrierInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	var list []models.CarrierInfo
	seen := make(map[string]bool)

	for key, c := range r.carriers {
		info := c.Info()
		if seen[info.Slug] {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(info.Slug), q) ||
			strings.Contains(strings.ToLower(key), q) ||
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
