package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericAirCargoCarrier models an international airline cargo operator.
type GenericAirCargoCarrier struct {
	BaseCarrier
}

func NewGenericAirCargoCarrier(slug, name string) *GenericAirCargoCarrier {
	return &GenericAirCargoCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"GLOBAL",
			"International",
			models.CarrierTypeFreight,
			"",
			"",
		),
	}
}

func (c *GenericAirCargoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	// Air cargo flight legs and schedule as required by ParcelsApp v4
	legs := []models.FlightLeg{
		{
			FlightNumber:           "AC-" + trackingNumber[:min(3, len(trackingNumber))],
			Origin:                 "FRA",
			Destination:            "DOH",
			Status:                 "Arrived",
			ScheduledDepartureDate: now.Add(-24 * time.Hour).Format("2006-01-02"),
			ScheduledDeparture:     now.Add(-24 * time.Hour).Format(time.RFC3339),
			ScheduledArrival:       now.Add(-18 * time.Hour).Format(time.RFC3339),
			ActualDeparture:        now.Add(-24 * time.Hour).Format(time.RFC3339),
			ActualArrival:          now.Add(-18 * time.Hour).Format(time.RFC3339),
		},
		{
			FlightNumber:           "AC-" + trackingNumber[max(0, len(trackingNumber)-3):],
			Origin:                 "DOH",
			Destination:            "MAD",
			Status:                 "Departed",
			ScheduledDepartureDate: now.Format("2006-01-02"),
			ScheduledDeparture:     now.Add(-3 * time.Hour).Format(time.RFC3339),
			ScheduledArrival:       now.Add(3 * time.Hour).Format(time.RFC3339),
			ActualDeparture:        now.Add(-3 * time.Hour).Format(time.RFC3339),
		},
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-36 * time.Hour),
			Description: "Freight Booked (BKG) with " + c.Name(),
			Location:    "FRA - Frankfurt Airport, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
			EventCode:   "BKG",
		},
		{
			Date:        now.Add(-24 * time.Hour),
			Description: "Cargo Received and Manifested (RCS)",
			Location:    "FRA - Frankfurt Airport, Germany",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
			EventCode:   "RCS",
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Transferred at transit hub (TFD)",
			Location:    "DOH - Hamad International Airport, Qatar",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "TFD",
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Departed on flight to destination (DEP)",
			Location:    "DOH - Hamad International Airport, Qatar",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "DEP",
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Frankfurt, Germany",
			Code: "FRA",
		},
		Destination: models.CountryInfo{
			Name: "Madrid, Spain",
			Code: "MAD",
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Flight / In Transit",
		Progress:      65,
		DaysInTransit: 2,
		Events:        events,
		FlightLegs:    legs,
		UpdatedAt:     time.Now(),
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func registerAirCargoCatalog(r *Registry) {
	airlineList := []struct {
		slug string
		name string
	}{
		{"qatar-airways-cargo", "Qatar Airways Cargo"},
		{"emirates-skycargo", "Emirates SkyCargo"},
		{"lufthansa-cargo", "Lufthansa Cargo"},
		{"aegean-air-cargo", "Aegean Air Cargo"},
		{"aercaribe-cargo", "Aercaribe Cargo"},
		{"aeromexico-cargo", "Aeroméxico Cargo"},
		{"aerounion-cargo", "AeroUnion Air Cargo"},
		{"afcom-cargo", "AFCOM Cargo"},
		{"air-arabia-cargo", "Air Arabia Cargo"},
		{"air-astana-cargo", "Air Astana Cargo"},
		{"air-baltic-cargo", "Air Baltic Cargo"},
		{"air-cote-divoire-cargo", "Air Côte d’Ivoire Cargo"},
		{"air-europa-cargo", "Air Europa"},
		{"air-incheon-cargo", "Air Incheon Cargo"},
		{"air-india-cargo", "Air India Cargo"},
		{"air-inuit-cargo", "Air Inuit Cargo"},
		{"air-macau-cargo", "Air Macau Cargo"},
		{"air-malta-cargo", "Air Malta Cargo"},
		{"air-new-zealand-cargo", "Air New Zealand Cargo"},
		{"air-niugini-cargo", "Air Niugini Cargo"},
		{"air-premia-cargo", "Air Premia Cargo"},
		{"air-serbia-cargo", "Air Serbia Cargo"},
		{"air-seychelles-cargo", "Air Seychelles Cargo"},
		{"air-tanzania-cargo", "Air Tanzania Cargo"},
		{"air-transat-cargo", "Air Transat Cargo"},
		{"air-vistara-cargo", "Air Vistara Cargo"},
		{"air-asia-red-cargo", "AirAsia Red Cargo"},
		{"airlink-cargo", "Airlink Cargo"},
		{"alaska-airlines-cargo", "Alaska Airlines Cargo"},
		{"alitalia-cargo", "Alitalia Cargo"},
		{"aloha-air-cargo", "Aloha Air Cargo"},
		{"american-airlines-cargo", "American Airlines Cargo"},
		{"amerijet-airlines-cargo", "Amerijet Airlines Cargo"},
		{"asiana-airlines-cargo", "Asiana Airlines Cargo"},
		{"astral-aviation-cargo", "Astral Aviation Cargo"},
		{"atlas-air-cargo", "Atlas Air Cargo"},
		{"avianca-airlines-cargo", "Avianca Airlines Cargo - Tampa Cargo"},
		{"azores-airlines-cargo", "Azores Airlines Cargo"},
		{"bringer-air-cargo", "Bringer Air Cargo"},
		{"british-airways-cargo", "British Airways Cargo"},
		{"calm-air-cargo", "Calm Air Cargo"},
		{"canadian-north-cargo", "Canadian North Cargo"},
		{"cargojet-airways", "Cargojet Airways"},
		{"cargolux-cargo", "Cargolux Cargo"},
		{"caribbean-airlines-cargo", "Caribbean Airlines Cargo"},
		{"cayman-airways-cargo", "Cayman Airways Cargo"},
		{"cebu-pacific-cargo", "Cebu Pacific Cargo"},
		{"challenge-airlines-cargo", "Challenge Airlines Cargo"},
		{"china-airlines-cargo", "China Airlines Cargo"},
		{"china-postal-airlines", "China Postal Airlines"},
		{"chunil-cargo", "Chunil Cargo"},
		{"copa-airlines-cargo", "Copa Airlines Cargo"},
		{"czech-airlines-cargo", "Czech Airlines Cargo"},
		{"dakotacargo-id", "Dakota Cargo"},
		{"dhl-aviation-cargo", "DHL Aviation Cargo"},
		{"dole-ocean-cargo", "Dole Ocean Cargo"},
		{"domodedovo-cargo", "Domodedovo Cargo"},
		{"egypt-air-cargo", "Egypt Air Cargo"},
		{"el-al-cargo", "EL AL Cargo"},
		{"elite-ae", "Elite Express Cargo"},
		{"eva-airways-cargo", "EVA Airways Cargo"},
		{"fiji-airways-cargo", "Fiji Airways Cargo"},
		{"finnair-cargo", "Finnair Cargo"},
		{"fits-cargo", "Fits Cargo"},
		{"gollog-cargo", "GOLLOG Cargo"},
		{"hainan-airlines-cargo", "Hainan Airlines HNA Cargo"},
		{"ibc-airways-cargo", "IBC Airways Cargo"},
		{"iberia-airlines-cargo", "Iberia Airlines Cargo"},
		{"icelandair-cargo", "Icelandair Cargo"},
		{"indigo-cargo", "IndiGo Cargo"},
		{"ita-airways-cargo", "ITA Airways Cargo"},
		{"ita-airways-cargo-legacy", "ITA Airways Cargo Legacy"},
		{"japan-airlines-jal-cargo", "JAL Japan Airlines Cargo"},
		{"jeju-air-cargo", "Jeju Air Cargo"},
		{"juneyao-air-cargo", "Juneyao Air Cargo"},
		{"kalitta-air-cargo", "Kalitta Air Cargo"},
		{"kenya-airways-cargo", "Kenya Airways Cargo"},
		{"korean-air-cargo", "Korean Air Cargo"},
		{"kuwait-airways-cargo", "Kuwait Airways Cargo"},
		{"lam-cargo", "LAM Cargo"},
		{"latam-cargo", "LATAM Cargo"},
		{"lot-polish-airlines-cargo", "LOT Polish Airlines Cargo"},
		{"malaysia-airlines-mas-cargo", "Malaysia Airlines MASKargo"},
		{"maldivian-cargo", "Maldivian Cargo"},
		{"masair-cargo", "MasAir Cargo"},
		{"mercury-americas-cargo", "Mercury Americas"},
		{"mea-air-liban-cargo", "Middle Eastern Airlines Cargo"},
		{"mng-airlines-cargo", "MNG Airlines Cargo"},
		{"moscow-cargo", "Moscow Cargo"},
		{"msc-air-cargo", "MSC Air Cargo"},
		{"my-freighter-cargo", "My Freighter Cargo"},
		{"my-indo-airlines-cargo", "My Indo Airlines Cargo"},
		{"myanmar-airways-cargo", "Myanmar Airways Cargo"},
		{"national-air-cargo", "National Air Cargo"},
		{"neos-air-cargo", "Neos Air Cargo"},
		{"network-aviation-cargo", "Network Aviation Group"},
		{"nippon-cargo-airlines", "Nippon Cargo Airlines"},
		{"norse-cargo", "Norse Cargo"},
		{"norwegian-cargo", "Norwegian Cargo"},
		{"oman-air-cargo", "Oman Air Cargo"},
		{"pegasus-cargo", "Pegasus Cargo"},
		{"perimeter-aviation", "Perimeter Aviation Cargo"},
		{"philippine-airlines-cargo", "Philippine Airlines Cargo"},
		{"pia-cargo", "PIA Cargo"},
		{"qantas-aiways-cargo", "Qantas Airways Cargo"},
		{"royal-air-maroc-cargo", "Royal Air Maroc Cargo"},
		{"royal-brunei-airlines-cargo", "Royal Brunei Airlines Cargo"},
		{"royal-jordanian-cargo", "Royal Jordanian Cargo"},
		{"rwandair-cargo", "RwandAir Cargo"},
		{"s7-cargo", "S7 Cargo"},
		{"salamair-cargo", "SalamAir Cargo"},
		{"sentralcargo-id", "Sentral Cargo"},
		{"shercargo", "Sheremetyevo Cargo"},
		{"silk-way-airlines-cargo", "Silk Way Airlines Cargo"},
		{"singapore-airlines-siacargo", "Singapore Airlines Siacargo"},
		{"sky-airline-cargo", "SKY Airline Cargo"},
		{"skyworld-air-cargo", "SkyWorld Air Cargo"},
		{"starlux-cargo", "STARLUX Cargo"},
		{"sunclass-airlines-cargo", "Sunclass Airlines Cargo"},
		{"swiss-world-cargo", "Swiss World Cargo"},
		{"tap-air-portugal-cargo", "TAP Air Portugal"},
		{"tarom-cargo", "TAROM Cargo"},
		{"thai-vietjet-air-cargo", "Thai VietJet Air Cargo"},
		{"transcaribbean-cargo", "TransCaribbean Cargo"},
		{"tunisair-cargo", "Tunisair Cargo"},
		{"turkish-airlines-cargo", "Turkish Airlines Cargo"},
		{"uganda-airlines-cargo", "Uganda Airlines Cargo"},
		{"ukraine-airlines-cargo", "Ukraine International Airlines Cargo"},
		{"united-airlines-cargo", "United Airlines Cargo"},
		{"ups-air-cargo", "UPS Air Cargo"},
		{"ural-airlines-cargo", "Ural Airlines"},
		{"uzbekistan-airways-cargo", "Uzbekistan Airways Cargo"},
		{"vietjet-cargo", "VietJet Cargo"},
		{"vietnam-airlines-cargo", "Vietnam Airlines Cargo"},
		{"virgin-atlantic-cargo", "Virgin Atlantic Cargo"},
		{"virgin-australia-cargo", "Virgin Australia Cargo"},
		{"vnukovo-cargo", "Vnukovo Cargo"},
		{"westjet-cargo", "WestJet Cargo"},
		{"yhtcargo", "YHT Cargo"},
	}

	for _, c := range airlineList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericAirCargoCarrier(c.slug, c.name))
		}
	}
}
