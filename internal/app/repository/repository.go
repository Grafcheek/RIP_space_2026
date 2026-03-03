package repository

import (
	"fmt"
	"math"
	"strings"
)

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

// TransferRoute — услуга: межпланетный перелёт «откуда → куда».
// Расчёты упрощены до гелиоцентрического перехода Гомана (круговые орбиты).
type TransferRoute struct {
	ID          int
	Title       string
	From        string
	To          string
	Description string
	Image       string  // ключ изображения в Minio (bucket/объект)
	Video       string  // ключ видео в Minio
	FromOrbitAU float64 // радиус орбиты отправителя (а.е.)
	ToOrbitAU   float64 // радиус орбиты получателя (а.е.)
}

// MissionProfile — заявка: исходные параметры аппарата и набора перелётов для расчёта.
type MissionProfile struct {
	ID          int
	Title       string
	Description string

	DryMassKg   float64 // сухая масса аппарата (кг)
	IspSeconds  float64 // удельный импульс (с)
	Routes      []MissionRoute
	RouteCount  int
}

// MissionRoute — связь м-м: маршрут + результаты расчёта.
type MissionRoute struct {
	Route       TransferRoute
	DeltaVms    float64 // характеристическая скорость, м/с
	PropellantKg float64 // требуемая масса топлива, кг (по Циолковскому)
	EnergyJ     float64 // оценка энергии, Дж (см. CalculateEnergyJ)
}

// GetRoutes возвращает набор межпланетных маршрутов.
func (r *Repository) GetRoutes() ([]TransferRoute, error) {
	routes := []TransferRoute{
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
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("массив маршрутов пуст")
	}

	return routes, nil
}

// GetRoute возвращает маршрут по ID.
func (r *Repository) GetRoute(id int) (TransferRoute, error) {
	routes, err := r.GetRoutes()
	if err != nil {
		return TransferRoute{}, err
	}

	for _, rt := range routes {
		if rt.ID == id {
			return rt, nil
		}
	}
	return TransferRoute{}, fmt.Errorf("маршрут не найден")
}

// SearchRoutes возвращает маршруты, содержащие подстроку в названии/планетах.
func (r *Repository) SearchRoutes(query string) ([]TransferRoute, error) {
	routes, err := r.GetRoutes()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return routes, nil
	}

	var result []TransferRoute
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
// Используем E = 0.5 * m0 * Δv^2, где m0 = dryMass + propellant.
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

func (r *Repository) buildMissionProfile(id int, title, description string, routeIDs []int, dryMassKg, ispSeconds float64) (MissionProfile, error) {
	routes, err := r.GetRoutes()
	if err != nil {
		return MissionProfile{}, err
	}

	rtMap := make(map[int]TransferRoute, len(routes))
	for _, rt := range routes {
		rtMap[rt.ID] = rt
	}

	var missionRoutes []MissionRoute
	for _, rid := range routeIDs {
		rt, ok := rtMap[rid]
		if !ok {
			continue
		}

		dv := CalculateHohmannDeltaVms(rt.FromOrbitAU, rt.ToOrbitAU)
		prop := CalculatePropellantKg(dryMassKg, dv, ispSeconds)
		energy := CalculateEnergyJ(dryMassKg, prop, dv)

		missionRoutes = append(missionRoutes, MissionRoute{
			Route:        rt,
			DeltaVms:     dv,
			PropellantKg: prop,
			EnergyJ:      energy,
		})
	}

	return MissionProfile{
		ID:          id,
		Title:       title,
		Description: description,
		DryMassKg:   dryMassKg,
		IspSeconds:  ispSeconds,
		Routes:      missionRoutes,
		RouteCount:  len(missionRoutes),
	}, nil
}

// GetMissionProfiles возвращает «заявки» на расчёт.
func (r *Repository) GetMissionProfiles() ([]MissionProfile, error) {
	profile, err := r.buildMissionProfile(
		1,
		"Расчёт Δv и топлива для межпланетного перелёта",
		"Параметры аппарата заданы фиксированно (Lab 1: без редактирования). Для каждого маршрута считаем Δv по переходу Гомана, массу топлива по Циолковскому и грубую оценку энергии.",
		[]int{1, 2, 3, 4},
		2_000, // сухая масса, кг
		320,   // Isp, с (условный химический двигатель)
	)
	if err != nil {
		return nil, err
	}
	return []MissionProfile{profile}, nil
}

// GetMissionProfile возвращает «заявку» по ID.
func (r *Repository) GetMissionProfile(id int) (MissionProfile, error) {
	profiles, err := r.GetMissionProfiles()
	if err != nil {
		return MissionProfile{}, err
	}
	for _, p := range profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return MissionProfile{}, fmt.Errorf("заявка не найдена")
}

// GetMissionRouteForRoute ищет запись м-м для маршрута внутри заявки (для детальной страницы).
func (r *Repository) GetMissionRouteForRoute(routeID int) (*MissionRoute, error) {
	profiles, err := r.GetMissionProfiles()
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		for _, mr := range p.Routes {
			if mr.Route.ID == routeID {
				return &mr, nil
			}
		}
	}
	return nil, fmt.Errorf("маршрут не найден в заявках")
}
