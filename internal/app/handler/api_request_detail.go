package handler

import (
	"strings"

	"web_backend/internal/app/repository"
)

// Публичный базовый URL объектов в MinIO (как в HTML-шаблонах: /spaceobjects/...).
const minioSpaceObjectsPublicBase = "http://localhost:9000/spaceobjects/"

// interplanetaryFlightRequestDetail — ответ GET /api/interplanetaryflightrequests/:id и успешного PUT .../form:
// без вложенного объекта «услуга»: все поля сегмента и перелёта в одном элементе items[].
type interplanetaryFlightRequestDetail struct {
	ID                  int                                      `json:"id"`
	Status              string                                   `json:"status"`
	CreatedAt           string                                   `json:"created_at"`
	FormedAt            *string                                  `json:"formed_at,omitempty"`
	CompletedAt         *string                                  `json:"completed_at,omitempty"`
	SpacecraftDryMassKg float64                                  `json:"spacecraft_dry_mass_kg"`
	EngineIspSec        float64                                  `json:"engine_isp_sec"`
	TotalFuelMassKg     *float64                                 `json:"total_fuel_mass_kg,omitempty"`
	Items               []interplanetaryFlightRequestLineFlatDTO `json:"items"`
}

type interplanetaryFlightRequestLineFlatDTO struct {
	RouteID      int `json:"route_id"`
	Quantity     int `json:"quantity"`
	SegmentOrder int `json:"segment_order"`
	IsPrimary    bool `json:"is_primary"`

	PayloadMassKg    *float64 `json:"payload_mass_kg,omitempty"`
	SegmentDryMassKg *float64 `json:"segment_dry_mass_kg,omitempty"`
	SegmentIspSec    *float64 `json:"segment_isp_sec,omitempty"`

	// Сохранённые при формировании заявки (лаб. 2 / FormRequest)
	StoredDeltaVMs   *float64 `json:"stored_delta_v_ms,omitempty"`
	StoredFuelMassKg *float64 `json:"stored_fuel_mass_kg,omitempty"`

	// Поля межпланетного перелёта (услуги) в том же объекте
	InterplanetaryFlightTitle       string  `json:"interplanetary_flight_title"`
	InterplanetaryFlightFrom        string  `json:"interplanetary_flight_from"`
	InterplanetaryFlightTo          string  `json:"interplanetary_flight_to"`
	InterplanetaryFlightDescription string  `json:"interplanetary_flight_description"`
	FromOrbitRadiusKm               float64 `json:"from_orbit_radius_km"`
	ToOrbitRadiusKm                 float64 `json:"to_orbit_radius_km"`
	ImageKey                        string  `json:"image_key"`
	VideoKey                        string  `json:"video_key"`
	ImageURL                        string  `json:"image_url"`
	VideoURL                        string  `json:"video_url"`

	// Текущий расчёт по формулам лаб. 2 (удобно и для черновика)
	DeltaVMs     float64 `json:"delta_v_ms"`
	PropellantKg float64 `json:"propellant_kg"`
	EnergyJ      float64 `json:"energy_j"`
}

func spaceObjectURL(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	return strings.TrimRight(minioSpaceObjectsPublicBase, "/") + "/" + key
}

func buildInterplanetaryFlightRequestDetail(fr *repository.FlightRequest) interplanetaryFlightRequestDetail {
	out := interplanetaryFlightRequestDetail{
		ID:                  fr.ID,
		Status:              fr.Status,
		CreatedAt:           fr.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		SpacecraftDryMassKg: fr.DryMassKg,
		EngineIspSec:        fr.IspSeconds,
		TotalFuelMassKg:     fr.TotalFuelMassKg,
		Items:               nil,
	}
	if fr.FormedAt != nil {
		s := fr.FormedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		out.FormedAt = &s
	}
	if fr.CompletedAt != nil {
		s := fr.CompletedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		out.CompletedAt = &s
	}

	out.Items = make([]interplanetaryFlightRequestLineFlatDTO, 0, len(fr.Items))

	for _, it := range fr.Items {
		route := it.Route
		mass := fr.DryMassKg
		if it.SegmentDryMassKg != nil {
			mass = *it.SegmentDryMassKg
		}
		isp := fr.IspSeconds
		if it.SegmentIspSec != nil {
			isp = *it.SegmentIspSec
		}
		dv := repository.CalculateDeltaVms(route.FromOrbitKm, route.ToOrbitKm)
		fuel := repository.CalculatePropellantKg(mass, dv, isp)
		energy := repository.CalculateEnergyJ(mass, fuel, dv)

		line := interplanetaryFlightRequestLineFlatDTO{
			RouteID:                         it.RouteID,
			Quantity:                        it.Quantity,
			SegmentOrder:                    it.SegmentOrder,
			IsPrimary:                       it.IsPrimary,
			PayloadMassKg:                   it.PayloadMassKg,
			SegmentDryMassKg:                it.SegmentDryMassKg,
			SegmentIspSec:                   it.SegmentIspSec,
			StoredDeltaVMs:                  it.DeltaVKms,
			StoredFuelMassKg:                it.FuelMassKg,
			InterplanetaryFlightTitle:       route.Title,
			InterplanetaryFlightFrom:        route.From,
			InterplanetaryFlightTo:          route.To,
			InterplanetaryFlightDescription: route.Description,
			FromOrbitRadiusKm:               route.FromOrbitKm,
			ToOrbitRadiusKm:                 route.ToOrbitKm,
			ImageKey:                        route.Image,
			VideoKey:                        route.Video,
			ImageURL:                        spaceObjectURL(route.Image),
			VideoURL:                        spaceObjectURL(route.Video),
			DeltaVMs:                        dv,
			PropellantKg:                    fuel,
			EnergyJ:                         energy,
		}
		out.Items = append(out.Items, line)
	}

	return out
}
