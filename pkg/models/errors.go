package models

import "errors"

// Canonical domain errors for tracking operations.
var (
	// ErrShipmentNotFound indicates the tracking number was not found or does not exist with the carrier.
	ErrShipmentNotFound = errors.New("shipment not found")

	// ErrCarrierNotImplemented indicates the carrier is recognized/catalogued but has no live network adapter.
	ErrCarrierNotImplemented = errors.New("carrier live tracking driver is not implemented")

	// ErrPostalCodeRequired indicates the carrier requires a destination postal code to perform tracking.
	ErrPostalCodeRequired = errors.New("postal code is required for this carrier")

	// ErrMissingPostalCode is an alias for ErrPostalCodeRequired for backward compatibility.
	ErrMissingPostalCode = ErrPostalCodeRequired

	// ErrCarrierRateLimited indicates the carrier API rate limit has been exceeded.
	ErrCarrierRateLimited = errors.New("carrier rate limit exceeded")

	// ErrCarrierTemporarilyUnavailable indicates the carrier service is down, throttled or returned a WAF challenge.
	ErrCarrierTemporarilyUnavailable = errors.New("carrier temporarily unavailable")

	// ErrScraperSchemaChanged indicates the carrier portal HTML/JSON structure changed and expected fields could not be parsed.
	ErrScraperSchemaChanged = errors.New("scraper schema changed")

	// ErrCarrierBlocked indicates the carrier portal blocked the request via WAF, CAPTCHA, or HTTP 403/429.
	ErrCarrierBlocked = errors.New("carrier blocked request")
)
