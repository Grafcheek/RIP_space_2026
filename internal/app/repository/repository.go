package repository

import (
	"fmt"
	"math"
	"strings"
)

// Соответствие сущностям ER (interplanetary flight):
//   InterplanetaryFlight            → interplanetary_flights (каталог перелётов / «услуга»).
//   InterplanetaryFlightRequest     → interplanetary_flights_requests (заявка).
//   InterplanetaryFlightInRequest   → interplanetary_flights_in_request (м-м заявка↔перелёт + поля связи и расчёт по сегменту).
// Lab 1: данные в памяти, без SQL.

// Repository — хранилище данных (Lab 1: данные в массивах, без БД).
type Repository struct{}

// NewRepository создаёт новый экземпляр репозитория.
func NewRepository() (*Repository, error) {
	return &Repository{}, nil
}

const (
	auMeters = 149_597_870_700.0 // 1 а.е. в метрах
	muSun    = 1.32712440018e20  // гравитационный параметр Солнца, м^3/с^2
	g0       = 9.80665           // стандартное ускорение свободного падения, м/с^2
)

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

// InterplanetaryFlightRequest — заявка на расчёт (сущность interplanetary_flights_requests в ER).
// Поля м-м по массам аппарата и ДУ + агрегированный результат расчёта (total_*).
type InterplanetaryFlightRequest struct {
	ID int // id заявки

	Title       string
	Description string

	SpacecraftDryMassKg float64 
	EngineMassKg        float64 
	IspSeconds          float64 

	TotalFuelMassKg float64 
	TotalDeltaVms     float64 

	FlightsInRequest []InterplanetaryFlightInRequest 
	RouteCount       int
}

// InterplanetaryFlightInRequest — связь м-м: заявка ↔ перелёт (таблица interplanetary_flights_in_request в ER).
// Содержит поля связи (порядок, quantity, …) и расчётные поля по сегменту (результат на строку).
type InterplanetaryFlightInRequest struct {
	Flight InterplanetaryFlight // FK → 

	SegmentOrder  int    
	Quantity      int     
	IsPrimary     bool    
	PayloadMassKg float64 

	DeltaVms     float64 
	PropellantKg float64 
	EnergyJ      float64 
}

// GetInterplanetaryFlights возвращает каталог межпланетных перелётов (interplanetary_flights).
func (r *Repository) GetInterplanetaryFlights() ([]InterplanetaryFlight, error) {
	routes := []InterplanetaryFlight{
		{
			ID:          1,
			Title:       "Юпитер",
			From:        "Земля",
			To:          "Юпитер",
			Description: "Упрощённый перелёт к газовому гиганту. Модель: переход Гомана вокруг Солнца (без гравитационных манёвров, без учёта наклонения орбит).",
			Image:       "Jupiter.jpeg",
			Video:       "Jupiter_vid.mp4",
			FromOrbitAU: 1.000,
			ToOrbitAU:   5.204,
		},
		{
			ID:          2,
			Title:       "Сатурн",
			From:        "Земля",
			To:          "Сатурн",
			Description: "Дальняя цель: перелёт к Сатурну. В реальности часто используют гравитационные манёвры (как у Voyager), но здесь считаем идеализированный переход Гомана.",
			Image:       "Saturn.jpg",
			Video:       "Saturn_vid.mp4",
			FromOrbitAU: 1.000,
			ToOrbitAU:   9.583,
		},
		{
			ID:          3,
			Title:       "Уран",
			From:        "Земля",
			To:          "Уран",
			Description: "Перелёт к ледяному гиганту (идеализированная гелиоцентрическая траектория).",
			Image:       "Uranus.jpg",
			Video:       "Uranus_vid.mp4",
			FromOrbitAU: 1.000,
			ToOrbitAU:   19.218,
		},
		{
			ID:          4,
			Title:       "Нептун",
			From:        "Земля",
			To:          "Нептун",
			Description: "Одна из самых «дорогих» целей по Δv в упрощённой модели. Voyager 2 долетел до Нептуна с помощью гравитационных манёвров — здесь их не учитываем.",
			Image:       "Neptune.jpg",
			Video:       "Neptune_vid.mp4",
			FromOrbitAU: 1.000,
			ToOrbitAU:   30.110,
		},
		// Обратные interplanetary flights: с планет обратно на Землю.
		{
			ID:          5,
			Title:       "Обратный с Юпитера",
			From:        "Юпитер",
			To:          "Земля",
			Description: "Обратный перелёт с орбиты Юпитера на Землю. Те же параметры орбит, но теперь считаем характеристическую скорость и топливо для пути домой.",
			Image:       "Earth.jpg",
			Video:       "Earth_vid.mp4",
			FromOrbitAU: 5.204,
			ToOrbitAU:   1.000,
		},
		{
			ID:          6,
			Title:       "Обратный с Сатурна",
			From:        "Сатурн",
			To:          "Земля",
			Description: "Обратный перелёт с орбиты Сатурна на Землю: моделируем возврат после глубокой внешней миссии и оцениваем Δv и массу топлива.",
			Image:       "Earth.jpg",
			Video:       "Earth_vid.mp4",
			FromOrbitAU: 9.583,
			ToOrbitAU:   1.000,
		},
		{
			ID:          7,
			Title:       "Обратный с Урана",
			From:        "Уран",
			To:          "Земля",
			Description: "Обратный перелёт с орбиты Урана на Землю. Ледяной гигант остаётся позади, а мы считаем Δv и топливо для возвращения к Земле.",
			Image:       "Earth.jpg",
			Video:       "Earth_vid.mp4",
			FromOrbitAU: 19.218,
			ToOrbitAU:   1.000,
		},
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

// CalculateHohmannDeltaVms считает суммарное Δv (м/с) для перехода Гомана между круговыми орбитами.
func CalculateHohmannDeltaVms(r1AU, r2AU float64) float64 {
	if r1AU <= 0 || r2AU <= 0 {
		return 0
	}
	r1 := r1AU * auMeters
	r2 := r2AU * auMeters

	v1 := math.Sqrt(muSun / r1)
	v2 := math.Sqrt(muSun / r2)
	a := (r1 + r2) / 2.0

	vt1 := math.Sqrt(muSun*(2.0/r1-1.0/a))
	vt2 := math.Sqrt(muSun*(2.0/r2-1.0/a))

	dv1 := math.Abs(vt1 - v1)
	dv2 := math.Abs(v2 - vt2)
	return dv1 + dv2
}

// CalculatePropellantKg оценивает массу топлива по формуле Циолковского (mf = dryMass).
func CalculatePropellantKg(dryMassKg, deltaVms, ispSeconds float64) float64 {
	if dryMassKg <= 0 || deltaVms <= 0 || ispSeconds <= 0 {
		return 0
	}
	return dryMassKg * (math.Exp(deltaVms/(ispSeconds*g0)) - 1.0)
}

// CalculateEnergyJ даёт простую оценку энергии на «разгон» под Δv (не орбитальная механика).
func CalculateEnergyJ(dryMassKg, propellantKg, deltaVms float64) float64 {
	if deltaVms <= 0 {
		return 0
	}
	m0 := dryMassKg + propellantKg
	if m0 <= 0 {
		return 0
	}
	return 0.5 * m0 * deltaVms * deltaVms
}

func (r *Repository) buildInterplanetaryFlightRequest(id int, title, description string, flightIDs []int, spacecraftDryMassKg, engineMassKg, ispSeconds float64) (InterplanetaryFlightRequest, error) {
	catalog, err := r.GetInterplanetaryFlights()
	if err != nil {
		return InterplanetaryFlightRequest{}, err
	}

	flightMap := make(map[int]InterplanetaryFlight, len(catalog))
	for _, f := range catalog {
		flightMap[f.ID] = f
	}

	var rows []InterplanetaryFlightInRequest
	var totalFuel, totalDv float64
	segmentOrder := 0

	for _, fid := range flightIDs {
		fl, ok := flightMap[fid]
		if !ok {
			continue
		}
		segmentOrder++

		dv := CalculateHohmannDeltaVms(fl.FromOrbitAU, fl.ToOrbitAU)
		prop := CalculatePropellantKg(spacecraftDryMassKg, dv, ispSeconds)
		energy := CalculateEnergyJ(spacecraftDryMassKg, prop, dv)
		totalFuel += prop
		totalDv += dv

		rows = append(rows, InterplanetaryFlightInRequest{
			Flight:        fl,
			SegmentOrder:  segmentOrder,
			Quantity:      1,
			IsPrimary:     segmentOrder == 1,
			PayloadMassKg: 0,
			DeltaVms:      dv,
			PropellantKg:  prop,
			EnergyJ:       energy,
		})
	}

	return InterplanetaryFlightRequest{
		ID:                  id,
		Title:               title,
		Description:         description,
		SpacecraftDryMassKg: spacecraftDryMassKg,
		EngineMassKg:        engineMassKg,
		IspSeconds:          ispSeconds,
		TotalFuelMassKg:     totalFuel,
		TotalDeltaVms:       totalDv,
		FlightsInRequest:    rows,
		RouteCount:          len(rows),
	}, nil
}

// GetInterplanetaryFlightRequests список заявок (interplanetary_flights_requests).
func (r *Repository) GetInterplanetaryFlightRequests() ([]InterplanetaryFlightRequest, error) {
	req, err := r.buildInterplanetaryFlightRequest(
		1,
		"Расчёт параметров межпланетного перелёта",
		"Заявка на расчёт характеристической скорости Δv и необходимого количества топлива для указанных масс космического аппарата и двигательной установки.",
		[]int{1, 5, 2, 6, 3, 7, 4},
		2_000,
		320,
		320,
	)
	if err != nil {
		return nil, err
	}
	return []InterplanetaryFlightRequest{req}, nil
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
