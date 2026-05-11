package repository

import (
	"fmt"
	"strings"
)

// Соответствие сущностям ER (interplanetary flight):
//   InterplanetaryFlight            → interplanetary_flights (каталог услуг / перелётов).
//   InterplanetaryFlightRequest     → interplanetary_flights_requests (заявка).
//   InterplanetaryFlightInRequest   → interplanetary_flights_in_request (m-m заявка↔услуга + расчёт по строке).
// Lab 1: данные в памяти, без SQL.

// Repository — хранилище данных (Lab 1: данные в массивах, без БД).
type Repository struct{}

// NewRepository создаёт новый экземпляр репозитория.
func NewRepository() (*Repository, error) {
	return &Repository{}, nil
}

// InterplanetaryFlight — каталог услуги «межпланетный перелёт» (сущность interplanetary_flights в ER).
// Расчёты упрощены до гелиоцентрического перехода Гомана (круговые орбиты).
type InterplanetaryFlight struct {
	ID          int
	Title       string
	From        string
	To          string
	Description string
	Image       string  // image_url / ключ в Minio
	Video       string  // ключ видео в Minio
	FromOrbitAU float64 // радиус орбиты отправителя (а.е.)
	ToOrbitAU   float64 // радиус орбиты получателя (а.е.)
}

// PlanetDataModel — словарная запись отдельной услуги.
type PlanetDataModel struct {
	ID            int
	Name          string
	Description   string
	IsActive      bool
	ImageURL      string // ключ изображения в Minio
	RequiredDelta float64
}

// InterplanetaryFlightDataModel — словарная запись заявки.
type InterplanetaryFlightDataModel struct {
	ID            int
	Status        string
	CreatedAt     string
	CreatedBy     int
	FormedAt      string
	ModeratedBy   int
	CompletedAt   string
	EngineMassKg  float64
	IspSeconds    float64
	TotalFuelMass float64
	TotalDelta    float64
}

// PlanetsInFlightDataModel — строка m-m внутри заявки.
type PlanetsInFlightDataModel struct {
	ID               int
	FlightID         int
	RequestID        int
	PayloadMass      float64
	SpacecraftMassKg float64
	Delta            float64

	// Явные поля m-m для показа в Lab1.
	SegmentOrder int
	Quantity     int
	IsPrimary    bool
	Comment      string
}

var (
	Planets = map[int]PlanetDataModel{
		1: {
			ID:            1,
			Name:          "Юпитер",
			Description:   "Газовый гигант. Дальняя внешняя миссия.",
			IsActive:      true,
			ImageURL:      "Jupiter.jpeg",
			RequiredDelta: 8800,
		},
		2: {
			ID:            2,
			Name:          "Сатурн",
			Description:   "Миссия к Сатурну (упрощённая модель).",
			IsActive:      true,
			ImageURL:      "Saturn.jpg",
			RequiredDelta: 9800,
		},
		3: {
			ID:            3,
			Name:          "Уран",
			Description:   "Перелёт к ледяному гиганту Урану.",
			IsActive:      true,
			ImageURL:      "Uranus.jpg",
			RequiredDelta: 10600,
		},
		4: {
			ID:            4,
			Name:          "Нептун",
			Description:   "Одна из самых дорогих миссий по суммарному Δv.",
			IsActive:      true,
			ImageURL:      "Neptune.jpg",
			RequiredDelta: 11300,
		},
	}

	planetOrder = []int{1, 2, 3, 4}

	interplanetaryFlight = map[int]InterplanetaryFlightDataModel{
		1: {
			ID:            1,
			Status:        "completed",
			CreatedAt:     "2026-04-25T10:30:00Z",
			CreatedBy:     1,
			FormedAt:      "2026-04-25T11:00:00Z",
			ModeratedBy:   2,
			CompletedAt:   "2026-04-25T11:40:00Z",
			EngineMassKg:  320,
			IspSeconds:    320,
			TotalFuelMass: 14800,
			TotalDelta:    40500,
		},
	}

	// PlanetsInFlight — М-М 
	PlanetsInFlight = map[int][]PlanetsInFlightDataModel{
		1: {
			{
				ID:               1,
				FlightID:         1,
				RequestID:        1,
				PayloadMass:      600,
				SpacecraftMassKg: 2600,
				Delta:            8800,
				SegmentOrder:     1,
				Quantity:         1,
				IsPrimary:        true,
				Comment:          "Основная цель миссии.",
			},
			{
				ID:               2,
				FlightID:         2,
				RequestID:        1,
				PayloadMass:      450,
				SpacecraftMassKg: 2450,
				Delta:            9800,
				SegmentOrder:     2,
				Quantity:         1,
				IsPrimary:        false,
				Comment:          "Дополнительный пролёт.",
			},
			{
				ID:               3,
				FlightID:         3,
				RequestID:        1,
				PayloadMass:      350,
				SpacecraftMassKg: 2350,
				Delta:            10600,
				SegmentOrder:     3,
				Quantity:         1,
				IsPrimary:        false,
				Comment:          "Расширение маршрута.",
			},
			{
				ID:               4,
				FlightID:         4,
				RequestID:        1,
				PayloadMass:      250,
				SpacecraftMassKg: 2250,
				Delta:            11300,
				SegmentOrder:     4,
				Quantity:         1,
				IsPrimary:        false,
				Comment:          "Финальная дальняя точка.",
			},
		},
	}
)

// InterplanetaryFlightRequest — заявка на расчёт (сущность interplanetary_flights_requests в ER).
// У заявки только собственные поля и итоговые результаты.
// Вложенный массив m-m со строками расчёта вынесен в FlightsInRequest.
type InterplanetaryFlightRequest struct {
	ID int // id заявки

	Title       string
	Description string

	EngineMassKg float64
	IspSeconds   float64

	TotalFuelMassKg float64
	TotalDeltaVms   float64

	FlightsInRequest []InterplanetaryFlightInRequest
	RouteCount       int
}

// InterplanetaryFlightInRequest — связь м-м: заявка ↔ перелёт (таблица interplanetary_flights_in_request в ER).
// Содержит поля связи (порядок, quantity, …) и расчётные поля по сегменту.
// Здесь же лежат повторяющиеся по строкам поля вроде массы аппарата и Δv.
type InterplanetaryFlightInRequest struct {
	Flight InterplanetaryFlight

	SegmentOrder     int
	Quantity         int
	IsPrimary        bool
	PayloadMassKg    float64
	SpacecraftMassKg float64

	DeltaVms     float64
	PropellantKg float64
	EnergyJ      float64
}

// GetInterplanetaryFlights возвращает каталог межпланетных перелётов (interplanetary_flights).
func (r *Repository) GetInterplanetaryFlights() ([]InterplanetaryFlight, error) {
	routes := make([]InterplanetaryFlight, 0, len(planetOrder))
	for _, id := range planetOrder {
		pl, ok := Planets[id]
		if !ok || !pl.IsActive {
			continue
		}
		videoKey := pl.ImageURL
		videoKey = strings.TrimSuffix(videoKey, ".jpeg")
		videoKey = strings.TrimSuffix(videoKey, ".jpg")
		videoKey = strings.TrimSuffix(videoKey, ".png")
		routes = append(routes, InterplanetaryFlight{
			ID:          pl.ID,
			Title:       pl.Name,
			From:        "Земля",
			To:          pl.Name,
			Description: pl.Description,
			Image:       pl.ImageURL,
			Video:       videoKey + "_vid.mp4",
			FromOrbitAU: 1.000,
			ToOrbitAU:   1.000,
		})
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("каталог interplanetary flights пуст")
	}

	return routes, nil
}

// GetInterplanetaryFlightByID возвращает перелёт по id (interplanetary_flights.id).
func (r *Repository) GetInterplanetaryFlightByID(id int) (InterplanetaryFlight, error) {
	routes, err := r.GetInterplanetaryFlights()
	if err != nil {
		return InterplanetaryFlight{}, err
	}

	for _, rt := range routes {
		if rt.ID == id {
			return rt, nil
		}
	}
	return InterplanetaryFlight{}, fmt.Errorf("interplanetary flight не найден")
}

// SearchInterplanetaryFlights фильтрует каталог по названию и планетам.
func (r *Repository) SearchInterplanetaryFlights(query string) ([]InterplanetaryFlight, error) {
	routes, err := r.GetInterplanetaryFlights()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return routes, nil
	}

	var result []InterplanetaryFlight
	for _, rt := range routes {
		if strings.Contains(strings.ToLower(rt.Title), q) ||
			strings.Contains(strings.ToLower(rt.From), q) ||
			strings.Contains(strings.ToLower(rt.To), q) {
			result = append(result, rt)
		}
	}
	return result, nil
}

// GetInterplanetaryFlightRequests список заявок (interplanetary_flights_requests).
func (r *Repository) GetInterplanetaryFlightRequests() ([]InterplanetaryFlightRequest, error) {
	catalog, err := r.GetInterplanetaryFlights()
	if err != nil {
		return nil, err
	}
	flightMap := make(map[int]InterplanetaryFlight, len(catalog))
	for _, fl := range catalog {
		flightMap[fl.ID] = fl
	}

	requestOrder := []int{1}
	requests := make([]InterplanetaryFlightRequest, 0, len(requestOrder))

	for _, requestID := range requestOrder {
		model, ok := interplanetaryFlight[requestID]
		if !ok {
			continue
		}

		rowsData := PlanetsInFlight[requestID]
		rows := make([]InterplanetaryFlightInRequest, 0, len(rowsData))
		for _, link := range rowsData {
			fl, exists := flightMap[link.FlightID]
			if !exists {
				continue
			}
			rows = append(rows, InterplanetaryFlightInRequest{
				Flight:           fl,
				SegmentOrder:     link.SegmentOrder,
				Quantity:         link.Quantity,
				IsPrimary:        link.IsPrimary,
				PayloadMassKg:    link.PayloadMass,
				SpacecraftMassKg: link.SpacecraftMassKg,
				DeltaVms:         link.Delta,
				PropellantKg:     link.PayloadMass * 2.0,
				EnergyJ:          link.Delta * 1000.0,
			})
		}

		requests = append(requests, InterplanetaryFlightRequest{
			ID:               model.ID,
			Title:            "Заявка межпланетного рейса #" + fmt.Sprintf("%d", model.ID),
			Description:      "У заявки свои поля, внутри неё вложен массив m-m interplanetary_flights_in_request со строками расчёта и повторяющимися полями по сегментам.",
			EngineMassKg:     model.EngineMassKg,
			IspSeconds:       model.IspSeconds,
			TotalFuelMassKg:  model.TotalFuelMass,
			TotalDeltaVms:    model.TotalDelta,
			FlightsInRequest: rows,
			RouteCount:       len(rows),
		})
	}
	return requests, nil
}

// GetInterplanetaryFlightRequest заявка по id (interplanetary_flights_requests.id).
func (r *Repository) GetInterplanetaryFlightRequest(id int) (InterplanetaryFlightRequest, error) {
	requests, err := r.GetInterplanetaryFlightRequests()
	if err != nil {
		return InterplanetaryFlightRequest{}, err
	}
	for _, req := range requests {
		if req.ID == id {
			return req, nil
		}
	}
	return InterplanetaryFlightRequest{}, fmt.Errorf("interplanetary flight request не найдена")
}

// GetInterplanetaryFlightInRequestForFlight строка м-м для перелёта внутри заявки (детальная страница interplanetary flight).
func (r *Repository) GetInterplanetaryFlightInRequestForFlight(flightID int) (*InterplanetaryFlightInRequest, error) {
	requests, err := r.GetInterplanetaryFlightRequests()
	if err != nil {
		return nil, err
	}
	for ri := range requests {
		req := &requests[ri]
		for j := range req.FlightsInRequest {
			if req.FlightsInRequest[j].Flight.ID == flightID {
				return &req.FlightsInRequest[j], nil
			}
		}
	}
	return nil, fmt.Errorf("interplanetary flight не найден в заявке")
}
