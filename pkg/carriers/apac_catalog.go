package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericAPACCarrier models an Asia-Pacific national or regional carrier.
type GenericAPACCarrier struct {
	BaseCarrier
}

func NewGenericAPACCarrier(slug, name string) *GenericAPACCarrier {
	return &GenericAPACCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"APAC",
			"Asia-Pacific",
			models.CarrierTypeLogistics,
			"",
			"",
		),
	}
}

func (c *GenericAPACCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Destination"
	destCode := "XX"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Parcel registered and picked up by " + c.Name(),
			Location:    "Origin Processing Center",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Processed at regional transit facility",
			Location:    "Regional Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-18 * time.Hour),
			Description: "Arrived at delivery distribution center",
			Location:    destCountry + " Local Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * time.Hour),
			Description: "Out for delivery with courier",
			Location:    destCountry + " Destination Area",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Asia-Pacific",
			Code: "APAC",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery",
		Progress:      85,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerAPACCatalog(r *Registry) {
	apacList := []struct {
		slug string
		name string
	}{
		{"australia-post", "Australia Post"},
		{"japan-post", "Japan Post"},
		{"2go", "2GO"},
		{"airpak-express", "Airpak Express"},
		{"airroad", "AirRoad"},
		{"gati-india", "AllCargo GATI"},
		{"allied-express-au", "Allied Express"},
		{"amazon-india-shipping", "Amazon India Shipping"},
		{"swiship-au", "Amazon Swiship AU"},
		{"swiship-jp", "Amazon Swiship JP"},
		{"anteraja-id", "AnterAja"},
		{"fastway-australia", "Aramex Australia"},
		{"fastway-new-zealand", "Aramex New Zealand"},
		{"arxp-au", "Australian Regional Express"},
		{"bangladesh-post", "Bangladesh Post"},
		{"best-express-my", "Best Express Malaysia"},
		{"bigpost-au", "BigPost"},
		{"bluedart", "Bluedart"},
		{"borderexpress-au", "Border Express"},
		{"call-courier-pk", "Call Courier"},
		{"cambodia-post", "Cambodia Post"},
		{"castleparcels-nz", "Castle Parcels"},
		{"centurion-au", "Centurion"},
		{"citilogistics", "Citilogistics"},
		{"cj-korea-express", "CJ Korea Express"},
		{"cj-logistics-korex", "CJ Korea Express International"},
		{"cj-logistics-my", "CJ Logistics Malaysia"},
		{"cj-logistics-sg", "CJ Logistics Singapore"},
		{"cj-logistics-th", "CJ Logistics Thailand"},
		{"codexpress-au", "CODExpress Australia"},
		{"couriers-please", "Couriers Please"},
		{"delhivery", "Delhivery"},
		{"delivere-au", "Delivere"},
		{"deliverybox-ph", "DeliveryBox"},
		{"dex-pk", "DEX PK"},
		{"direct-couriers-au", "Direct Couriers"},
		{"direct-freight-express-australia", "Direct Freight Express"},
		{"dragonfly-shipping", "Dragonfly Shipping Australia"},
		{"dtdc", "DTDC"},
		{"dtdc-india", "DTDC India"},
		{"echindia", "Echindia"},
		{"ecohai-jp", "Ecohai"},
		{"ekart", "Ekart Logistics"},
		{"ems-vietnam-post", "EMS Vietnam Post"},
		{"eparcel-kr", "eParcel.kr"},
		{"fast-courier-au", "Fast Courier Australia"},
		{"fastbox-kr", "Fastbox"},
		{"fastway-south-africa", "Fastway South Africa"},
		{"fdm-au", "FDM"},
		{"first-global-nz", "First Global Logistics"},
		{"flash-express", "Flash Express"},
		{"flash-express-ph", "Flash Express Philippines"},
		{"fliway-nz", "Fliway"},
		{"fstexpress-au", "FST Express"},
		{"fukutsu-jp", "Fukuyama Transporting (福山通運)"},
		{"gati", "GATI"},
		{"ghn-vn", "GHN / Giao Hang Nhanh"},
		{"gopeople-au", "GoPeople"},
		{"gtl-id", "GoTo Logistics"},
		{"hunter-express", "Hunter Express"},
		{"iel-th", "IEL"},
		{"ikea-indonesia", "IKEA Indonesia"},
		{"india-post", "India Post"},
		{"jaskipin-id", "Jaskipin"},
		{"kerry-express", "Kerry Express"},
		{"kerry-express-th", "Kerry Express Thailand"},
		{"kurasi-id", "Kurasi"},
		{"epantos-kr", "LX Pantos / ePantos"},
		{"mailplus-au", "Mailplus"},
		{"pos-malaysia", "Malaysia Post"},
		{"micro-express-au", "Micro Express"},
		{"new-zealand-post", "New Zealand Post"},
		{"ninja-van", "Ninja Van"},
		{"ninja-van-indonesia", "Ninja Van Indonesia"},
		{"ninja-van-philippines", "Ninja Van Philippines"},
		{"ninja-van-singapore", "Ninja Van Singapore"},
		{"ninja-van-thailand", "Ninja Van Thailand"},
		{"ninja-van-vietnam", "Ninja Van Vietnam"},
		{"one-world-courier-au", "One World Courier"},
		{"packsend-au", "Pack Send"},
		{"passtheparcel-nz", "PassTheParcel"},
		{"pbt-nz", "PBT Couriers"},
		{"philippines-post", "Philippines Post"},
		{"pickupp-my", "Pickupp Malaysia"},
		{"pickupp-sg", "Pickupp Singapore"},
		{"pos-indonesia", "POS Indonesia"},
		{"posthaste-nz", "Post Haste"},
		{"postex-pk", "PostEx"},
		{"raf-ph", "RAF Philippines"},
		{"rpx-id", "RPX Indonesia"},
		{"sagawa-global", "Sagawa Global"},
		{"sagawa-jp", "Sagawa Japan"},
		{"sagawa-global-ref", "Sagawa Reference"},
		{"sap-express-id", "SAP Express"},
		{"seino-jp", "Seino"},
		{"shadowfax-in", "ShadowFax"},
		{"hub-ez", "ShipGlobal India"},
		{"shipglobal-in", "ShipGlobal India"},
		{"shiprocket-in", "Shiprocket"},
		{"shipter-kr", "Shipter"},
		{"shopee-express-id", "Shopee Express Indonesia"},
		{"shopee-express-my", "Shopee Express Malaysia"},
		{"shopee-express-ph", "Shopee Express Philippines"},
		{"shopee-express-sg", "Shopee Express Singapore"},
		{"shopee-express-tw", "Shopee Express Taiwan"},
		{"shopee-express-th", "Shopee Express Thailand"},
		{"shopee-express-vn", "Shopee Express Vietnam"},
		{"singapore-post", "Singapore Post"},
		{"skynet-my", "SkyNet Malaysia"},
		{"south-korea-customs", "South Korea Customs Unipass"},
		{"korea-post", "South Korea Post"},
		{"speedpost-singapore", "Speedpost - Singapore Post"},
		{"t-cat-taiwan", "T-Cat Taiwan"},
		{"taqbin-my", "TA-Q-BIN Malaysia"},
		{"tcs-pk", "TCS Express"},
		{"tdtruck-nz", "TD Truck"},
		{"team-global-express-nz", "Team Global Express NZ"},
		{"thailand-post", "Thailand Post"},
		{"tnt-australia", "TNT Australia"},
		{"toll-ipec", "Toll IPEC"},
		{"toplogistics-au", "Top Logistics"},
		{"wwwexpress-ph", "W Express Cargo Philippines"},
		{"yamato-america", "Yamato America"},
		{"yamato-japan", "Yamato Japan"},
		{"zip-ph", "Zip"},
	}

	for _, c := range apacList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericAPACCarrier(c.slug, c.name))
		}
	}
}
