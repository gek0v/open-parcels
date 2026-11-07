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

// LiveTracker is implemented by carrier drivers that connect to real external APIs/networks.
type LiveTracker interface {
	HasLiveTracking() bool
}

// HasLiveTracking checks whether a carrier adapter supports real live network tracking.
func HasLiveTracking(c Carrier) bool {
	if c == nil {
		return false
	}
	if lt, ok := c.(LiveTracker); ok {
		return lt.HasLiveTracking()
	}
	if ac, ok := c.(*AliasCarrier); ok {
		return HasLiveTracking(ac.Carrier)
	}
	return false
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

// AliasCarrier wraps a Carrier under an alternative slug.
type AliasCarrier struct {
	Carrier
	slug string
}

func (a *AliasCarrier) Slug() string { return a.slug }

func registerDefaultCarriers(r *Registry) {
	r.Register(NewCainiaoCarrier())
	r.Register(NewCorreosSpainCarrier())

	correosExpress := NewCorreosExpressCarrier()
	r.Register(correosExpress)
	r.RegisterAlias("correos-express", correosExpress)

	r.Register(NewCTTExpressCarrier())
	r.Register(NewDHLExpressCarrier())

	dhlParcel := NewDHLParcelCarrier()
	r.Register(dhlParcel)
	r.RegisterAlias("dhl-parcel", dhlParcel)

	r.Register(NewDHLParcelSpainCarrier())
	r.Register(NewFedExCarrier())
	fpx := NewFourPXCarrier()
	r.Register(fpx)
	r.RegisterAlias("four-px", fpx)

	gls := NewGLSCarrier()
	r.Register(gls)
	r.RegisterAlias("gls", gls)

	r.Register(NewGLSSpainCarrier())

	inpost := NewInPostCarrier()
	r.Register(inpost)
	r.RegisterAlias("inpost", inpost)

	r.Register(NewInPostSpainCarrier())

	mrw := NewMRWCarrier()
	r.Register(mrw)
	r.RegisterAlias("mrw", mrw)
	r.Register(NewPaackCarrier())
	r.Register(NewSEURCarrier())
	r.Register(NewUPSCarrier())
	r.Register(NewUSPSCarrier())
	r.Register(NewYunExpressCarrier())
	r.Register(NewRoyalMailCarrier())
	r.Register(NewColissimoCarrier())
	r.Register(NewAirCargoCarrier())
	r.Register(NewOceanContainerCarrier())
	// Register Benelux regional catalog (18 carriers - Issue #16)
	registerBeneluxCatalog(r)

	// Register global ocean freight & container lines catalog (49 carriers - Issue #15)
	registerOceanFreightCatalog(r)

	// Register LATAM regional catalog (25 carriers - Issue #14)
	registerLATAMCatalog(r)

	// Register DACH region catalog (26 carriers - Issue #13)
	registerDACHCatalog(r)

	// Register Middle East & Africa regional catalog (57 carriers - Issue #12)
	registerMENAAfricaCatalog(r)

	// Register UK and Ireland regional catalog (38 carriers - Issue #11)
	registerUKIrelandCatalog(r)

	// Register France and Monaco regional catalog (22 carriers - Issue #10)
	registerFranceCatalog(r)

	// Register global air cargo catalog (136 airlines - Issue #8)
	registerAirCargoCatalog(r)

	// Register Nordic and Baltic regional catalog (23 carriers - Issue #6)
	registerNordicBalticCatalog(r)

	// Register North America regional catalog (42 carriers - Issue #5)
	registerNorthAmericaCatalog(r)

	// Register the Central & Eastern Europe and Balkans regional catalog (81 carriers - Issue #4)
	registerCEECatalog(r)

	// Register the Asia-Pacific regional catalog (125 carriers - Issue #3)
	registerAPACCatalog(r)

	// Register the global cross-border and multinational catalog (595 carriers - Issue #2)
	registerGlobalCrossBorderCatalog(r)

	// China & Hong Kong flagship carriers (Issue #1)
	r.Register(NewChinaPostCarrier())
	r.Register(NewSFExpressCarrier())
	r.Register(NewHongKongPostCarrier())
	r.Register(NewYanwenCarrier())
	r.Register(NewSunYouCarrier())
	r.Register(NewCNECarrier())
	r.Register(NewDepponCarrier())
	r.Register(NewYTOExpressCarrier())
	// Register the full China & Hong Kong regional catalog (271 carriers)
	registerChinaHongKongCatalog(r)

	// Italy & San Marino carriers (Issue #9)
	r.Register(NewBRTBartoliniCarrier())
	r.Register(NewBRTItalyParcelIDCarrier())
	r.Register(NewMyBRTCarrier())
	r.Register(NewPosteItalianeCarrier())
	r.Register(NewGLSItalyCarrier())
	r.Register(NewInPostItalyCarrier())
	r.Register(NewTNTItalyCarrier())
	r.Register(NewAmazonItalyCarrier())
	r.Register(NewSailpostCarrier())
	r.Register(NewAlfredITCarrier())
	r.Register(NewArcoSpedizioniCarrier())
	r.Register(NewCEVAITCarrier())
	r.Register(NewBDMITCarrier())
	r.Register(NewDeliverITCarrier())
	r.Register(NewLiccardiITCarrier())
	r.Register(NewMBEITCarrier())
	r.Register(NewRhenusITCarrier())
	r.Register(NewSpediamoITCarrier())
	r.Register(NewUnieuroITCarrier())

	// Spanish carriers (Issue #7)
	r.Register(NewActivos24Carrier())
	r.Register(NewAsendiaSpainCarrier())
	r.Register(NewAsignaCarrier())
	r.Register(NewCeleritasCarrier())
	r.Register(NewEnvialiaCarrier())
	r.Register(NewInstapackCarrier())
	r.Register(NewMBESpainCarrier())
	r.Register(NewNacexCarrier())
	r.Register(NewOntimeCarrier())
	r.Register(NewPacklinkCarrier())
	r.Register(NewRedurSpainCarrier())
	r.Register(NewRedurSpainRefCarrier())
	r.Register(NewRedurTrackingCarrier())
	r.Register(NewSendingCarrier())
	r.Register(NewTipsaCarrier())
	r.Register(NewZelerisCarrier())

	// Portugal carriers (Issue #17)
	r.Register(NewDelnextCarrier())
	r.Register(NewDPDPortugalCarrier())
	r.Register(NewPortugalPostCarrier())
	r.Register(NewWePickUpCarrier())
}

// Register adds a carrier to the registry.
func (r *Registry) Register(c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carriers[strings.ToLower(c.Slug())] = c
}

// RegisterAlias registers an alternative slug pointing to the same Carrier.
func (r *Registry) RegisterAlias(alias string, c Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carriers[strings.ToLower(alias)] = c
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

	for _, c := range r.carriers {
		if seen[c.Slug()] {
			continue
		}
		seen[c.Slug()] = true
		info := c.Info()
		if q == "" || strings.Contains(strings.ToLower(info.Slug), q) ||
			strings.Contains(strings.ToLower(info.Name), q) ||
			strings.Contains(strings.ToLower(info.Country), q) ||
			strings.Contains(strings.ToLower(info.CountryName), q) {
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

func (b BaseCarrier) Slug() string                   { return b.slug }
func (b BaseCarrier) Name() string                   { return b.name }
func (b BaseCarrier) Country() string                { return b.country }
func (b BaseCarrier) CountryName() string            { return b.countryName }
func (b BaseCarrier) Type() models.CarrierType       { return b.carrierType }
func (b BaseCarrier) Website() string                { return b.website }
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
