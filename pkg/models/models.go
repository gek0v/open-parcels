package models

import (
	"errors"
	"time"
)

// Sentinel errors for tracking operations
var (
	ErrShipmentNotFound      = errors.New("shipment not found")
	ErrMissingPostalCode     = errors.New("postal code is required for this carrier")
	ErrCarrierNotImplemented = errors.New("carrier live tracking driver is not implemented")
)

// Standard shipment statuses
type Status string

const (
	StatusPending        Status = "pending"
	StatusInfoReceived   Status = "info_received"
	StatusInTransit      Status = "in_transit"
	StatusOutForDelivery Status = "out_for_delivery"
	StatusDelivered      Status = "delivered"
	StatusFailedAttempt  Status = "failed_attempt"
	StatusException      Status = "exception"
	StatusExpired        Status = "expired"
)

// IsTerminal returns true if the shipment status is final.
func (s Status) IsTerminal() bool {
	return s == StatusDelivered || s == StatusExpired
}

// Stage represents the current phase in the transit lifecycle.
type Stage string

const (
	StageCreated   Stage = "created"
	StagePickedUp  Stage = "picked_up"
	StageTransit   Stage = "transit"
	StageCustoms   Stage = "customs"
	StageHandover  Stage = "handover"
	StageLastMile  Stage = "last_mile"
	StageDelivery  Stage = "delivery"
	StageException Stage = "exception"
)

// CarrierType categorizes the carrier service model.
type CarrierType string

const (
	CarrierTypePostal     CarrierType = "postal"
	CarrierTypeExpress    CarrierType = "express"
	CarrierTypeLogistics  CarrierType = "logistics"
	CarrierTypeFreight    CarrierType = "freight"
	CarrierTypeCrossBorder CarrierType = "cross_border"
)

// CarrierInfo provides metadata about a logistics carrier.
type CarrierInfo struct {
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Country     string      `json:"country,omitempty"`     // ISO-2 country code (e.g. "ES", "CN", "US", "GLOBAL")
	CountryName string      `json:"country_name,omitempty"`// e.g. "Spain", "China"
	Type        CarrierType `json:"type,omitempty"`
	Website     string      `json:"website,omitempty"`
	TrackingURL string      `json:"tracking_url,omitempty"`// Template or base URL
	Phone       string      `json:"phone,omitempty"`
	Logo        string      `json:"logo,omitempty"`
}

// CountryInfo represents a country involved in the shipment.
type CountryInfo struct {
	Name string `json:"name,omitempty"`
	Code string `json:"code,omitempty"` // ISO-2
}

// TrackingEvent is a normalized milestone in the shipment journey.
type TrackingEvent struct {
	Date         time.Time `json:"date"`
	RawDate      string    `json:"raw_date,omitempty"`
	Status       Status    `json:"status"`
	Stage        Stage     `json:"stage,omitempty"`
	Description  string    `json:"description"`
	Location     string    `json:"location,omitempty"`
	CarrierSlug  string    `json:"carrier_slug"`
	CarrierName  string    `json:"carrier_name"`
	EventCode    string    `json:"event_code,omitempty"`
	CarrierIndex int       `json:"carrier_index,omitempty"`
}

// FlightLeg models air cargo movements.
type FlightLeg struct {
	FlightNumber           string `json:"flight_number,omitempty"`
	Origin                 string `json:"origin,omitempty"`
	Destination            string `json:"destination,omitempty"`
	Status                 string `json:"status,omitempty"`
	ScheduledDepartureDate string `json:"scheduled_departure_date,omitempty"`
	ScheduledDeparture     string `json:"scheduled_departure,omitempty"`
	ScheduledArrival       string `json:"scheduled_arrival,omitempty"`
	ActualDeparture        string `json:"actual_departure,omitempty"`
	ActualArrival          string `json:"actual_arrival,omitempty"`
}

// Shipment is the complete unified view of a tracked parcel.
type Shipment struct {
	TrackingNumber           string          `json:"tracking_number"`
	Carrier                  CarrierInfo     `json:"carrier"`
	AdditionalCarriers       []CarrierInfo   `json:"additional_carriers,omitempty"`
	LastMileTrackingNumber   string          `json:"last_mile_tracking_number,omitempty"`
	LastMileCarrier          *CarrierInfo    `json:"last_mile_carrier,omitempty"`
	Status                   Status          `json:"status"`
	StatusText               string          `json:"status_text"`
	Progress                 int             `json:"progress"` // 0 to 100%
	Origin                   CountryInfo     `json:"origin,omitempty"`
	Destination              CountryInfo     `json:"destination,omitempty"`
	EstimatedDelivery        *time.Time      `json:"estimated_delivery,omitempty"`
	DeliveredAt              *time.Time      `json:"delivered_at,omitempty"`
	Events                   []TrackingEvent `json:"events"`
	DaysInTransit            int             `json:"days_in_transit"`
	Weight                   string          `json:"weight,omitempty"`
	ServiceType              string          `json:"service_type,omitempty"`
	FlightLegs               []FlightLeg     `json:"flight_legs,omitempty"`
	FromCache                bool            `json:"from_cache"`
	UpdatedAt                time.Time       `json:"updated_at"`
	Error                    string          `json:"error,omitempty"`
}

// DetectResult represents a candidate carrier match from the detector engine.
type DetectResult struct {
	Carrier        CarrierInfo `json:"carrier"`
	Confidence     float64     `json:"confidence"` // 0.0 to 1.0
	MatchedPattern string      `json:"matched_pattern,omitempty"`
	Reason         string      `json:"reason,omitempty"`
}

// TrackOptions configures a tracking request.
type TrackOptions struct {
	CarrierHint        string            `json:"carrier_hint,omitempty"`
	Language           string            `json:"language,omitempty"` // "es", "en" (default: "en")
	DestinationCountry string            `json:"destination_country,omitempty"`
	PostalCode         string            `json:"postal_code,omitempty"`
	AllowSimulated     bool              `json:"allow_simulated,omitempty"`
	Email              string            `json:"email,omitempty"`
	Phone              string            `json:"phone,omitempty"`
	AdditionalFields   map[string]string `json:"additional_fields,omitempty"`
	ForceRefresh       bool              `json:"force_refresh,omitempty"`
}

// ParcelsApp v4 API compatibility response types
type ParcelsAppTrackingRequest struct {
	Language        string                 `json:"language,omitempty"`
	WebhookURL      string                 `json:"webhook_url,omitempty"`
	Shipments       []ParcelsAppShipmentIn `json:"shipments"`
}

type ParcelsAppShipmentIn struct {
	TrackingNumber     string            `json:"tracking_number"`
	DestinationCountry string            `json:"destination_country,omitempty"`
	CarrierHint        string            `json:"carrier_hint,omitempty"`
	PostalCode         string            `json:"postal_code,omitempty"`
	AllowSimulated     bool              `json:"allow_simulated,omitempty"`
	Email              string            `json:"email,omitempty"`
	Phone              string            `json:"phone,omitempty"`
	AdditionalFields   map[string]string `json:"additional_fields,omitempty"`
}

type ParcelsAppResponse struct {
	RequestID string                  `json:"request_id"`
	Status    string                  `json:"status"` // "completed" | "processing"
	Done      bool                    `json:"done"`
	ExpiresAt string                  `json:"expires_at"`
	Results   []ParcelsAppResultItem  `json:"results"`
	Rejected  []ParcelsAppRejected    `json:"rejected,omitempty"`
}

type ParcelsAppResultItem struct {
	TrackingNumber string              `json:"tracking_number"`
	State          string              `json:"state"` // "completed" | "processing"
	FromCache      bool                `json:"from_cache"`
	Shipment       *ParcelsAppShipment `json:"shipment,omitempty"`
}

type ParcelsAppShipment struct {
	TrackingID       string                    `json:"trackingId"`
	Status           string                    `json:"status"`
	Origin           string                    `json:"origin,omitempty"`
	Destination      string                    `json:"destination,omitempty"`
	OriginCode       string                    `json:"originCode,omitempty"`
	DestinationCode  string                    `json:"destinationCode,omitempty"`
	States           []ParcelsAppStateItem     `json:"states"`
	Services         []ParcelsAppServiceItem   `json:"services"`
	DetectedCarrier  ParcelsAppServiceItem     `json:"detectedCarrier"`
	Detected         []int                     `json:"detected"`
	DaysInTransit    int                       `json:"daysInTransit,omitempty"`
	Attributes       []ParcelsAppAttribute     `json:"attributes,omitempty"`
	ExternalTracking []ParcelsAppExternalTrack `json:"externalTracking,omitempty"`
	FlightLegs       []FlightLeg               `json:"flight_legs,omitempty"`
	Error            string                    `json:"error,omitempty"`
}

type ParcelsAppStateItem struct {
	State    string `json:"state"`
	Location string `json:"location,omitempty"`
	Date     string `json:"date"`
	Carrier  int    `json:"carrier"`
}

type ParcelsAppServiceItem struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type ParcelsAppAttribute struct {
	Label string `json:"l"`
	Val   string `json:"val"`
	Code  string `json:"code,omitempty"`
}

type ParcelsAppExternalTrack struct {
	Slug       string `json:"slug"`
	URL        string `json:"url"`
	TrackingID string `json:"trackingId"`
	Method     string `json:"method"`
}

type ParcelsAppRejected struct {
	Index          int    `json:"index"`
	TrackingNumber string `json:"tracking_number,omitempty"`
	Code           string `json:"code"`
	Message        string `json:"message"`
}
