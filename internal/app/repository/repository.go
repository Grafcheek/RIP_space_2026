package repository

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository — слой доступа к данным (PostgreSQL + ORM).
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &Repository{db: db}, nil
}

const (
	muSun = 1.327e20 // гравитационный параметр Солнца, м^3/с^2 (1.327e11 км^3/с^2)
	g0    = 9.80665  // м/с^2
)

// TransferRoute — услуга: межпланетный перелёт «Земля → планета».
type TransferRoute struct {
	ID          int     `gorm:"column:id;primaryKey"`
	Title       string  `gorm:"column:title"`
	From        string  `gorm:"column:from_body"`
	To          string  `gorm:"column:to_body"`
	Description string  `gorm:"column:description"`
	IsDeleted   bool    `gorm:"column:is_deleted"`
	Image       string  `gorm:"column:image_url"`
	Video       string  `gorm:"column:video_url"`
	FromOrbitKm float64 `gorm:"column:from_orbit_radius_km"`
	ToOrbitKm   float64 `gorm:"column:to_orbit_radius_km"`
}

func (TransferRoute) TableName() string { return "transfer_routes" }

// FlightRequest — заявка на расчёты.
type FlightRequest struct {
	ID              int                     `gorm:"column:id;primaryKey"`
	Status          string                  `gorm:"column:status"`
	CreatedAt       time.Time               `gorm:"column:created_at"`
	CreatedBy       int                     `gorm:"column:created_by"`
	FormedAt        *time.Time              `gorm:"column:formed_at"`
	CompletedAt     *time.Time              `gorm:"column:completed_at"`
	ModeratedBy     *int                    `gorm:"column:moderated_by"`
	DryMassKg       float64                 `gorm:"column:spacecraft_dry_mass_kg"`
	IspSeconds      float64                 `gorm:"column:engine_isp_sec"`
	TotalFuelMassKg *float64                `gorm:"column:total_fuel_mass_kg"`
	Items           []FlightRequestRouteRow `gorm:"foreignKey:FlightRequestID;references:ID"`
}

func (FlightRequest) TableName() string { return "flight_requests" }

// FlightRequestRouteRow — м‑м заявка–услуга (составной ключ).
type FlightRequestRouteRow struct {
	FlightRequestID int `gorm:"column:flight_request_id;primaryKey"`
	RouteID         int `gorm:"column:route_id;primaryKey"`

	Quantity     int  `gorm:"column:quantity"`
	SegmentOrder int  `gorm:"column:segment_order"`
	IsPrimary    bool `gorm:"column:is_primary"`

	PayloadMassKg    *float64 `gorm:"column:payload_mass_kg"`
	DeltaVKms        *float64 `gorm:"column:delta_v_kms"`
	FuelMassKg       *float64 `gorm:"column:fuel_mass_kg"`
	SegmentDryMassKg *float64 `gorm:"column:segment_dry_mass_kg"`
	SegmentIspSec    *float64 `gorm:"column:segment_isp_sec"`

	Route TransferRoute `gorm:"foreignKey:RouteID;references:ID"`
}

func (FlightRequestRouteRow) TableName() string { return "flight_request_routes" }

// User — упрощённая модель пользователя для домена регистрации/аутентификации.
type User struct {
	ID           int64  `gorm:"column:id;primaryKey"`
	Username     string `gorm:"column:username"`
	PasswordHash string `gorm:"column:password_hash"`
	IsModerator  bool   `gorm:"column:is_moderator"`
}

func (User) TableName() string { return "users" }

// MissionProfile — DTO для шаблонов (как в Lab 1), собирается из БД.
type MissionProfile struct {
	ID          int
	Title       string
	Description string

	DryMassKg  float64
	IspSeconds float64

	Routes     []MissionRoute
	RouteCount int
	CanDelete  bool
}

type MissionRoute struct {
	Route            TransferRoute
	DeltaVms         float64
	PropellantKg     float64
	EnergyJ          float64
	SegmentDryMassKg float64
	SegmentIspSec    float64
}

// SearchRoutes (ORM): список услуг + серверный поиск.
func (r *Repository) SearchRoutes(query string) ([]TransferRoute, error) {
	var routes []TransferRoute

	q := strings.TrimSpace(query)
	tx := r.db.Where("is_deleted = false")
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			r.db.Where("title ILIKE ?", like).
				Or("from_body ILIKE ?", like).
				Or("to_body ILIKE ?", like),
		)
	}

	if err := tx.Order("id ASC").Find(&routes).Error; err != nil {
		return nil, err
	}
	return routes, nil
}

func (r *Repository) GetRoute(id int) (TransferRoute, error) {
	var rt TransferRoute
	if err := r.db.Where("id = ? AND is_deleted = false", id).First(&rt).Error; err != nil {
		return TransferRoute{}, err
	}
	return rt, nil
}

// CreateRoute создаёт новую услугу (маршрут) через ORM.
func (r *Repository) CreateRoute(rt *TransferRoute) error {
	return r.db.Create(rt).Error
}

func (r *Repository) GetDraft(userID int) (*FlightRequest, error) {
	var fr FlightRequest
	err := r.db.Preload("Items.Route").
		Where("created_by = ? AND status = 'draft'", userID).
		First(&fr).Error
	if err != nil {
		return nil, err
	}
	return &fr, nil
}

func (r *Repository) GetMission(userID, missionID int) (*FlightRequest, error) {
	var fr FlightRequest
	err := r.db.Preload("Items.Route").
		Where("id = ? AND created_by = ? AND status <> 'deleted'", missionID, userID).
		First(&fr).Error
	if err != nil {
		return nil, err
	}
	return &fr, nil
}

func (r *Repository) AddRouteToDraft(userID, routeID int) (*FlightRequest, error) {
	now := time.Now()

	var out FlightRequest
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var draft FlightRequest
		if err := tx.Where("created_by = ? AND status = 'draft'", userID).First(&draft).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				draft = FlightRequest{
					Status:     "draft",
					CreatedAt:  now,
					CreatedBy:  userID,
					DryMassKg:  2000,
					IspSeconds: 350,
				}
				if err := tx.Create(&draft).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		var maxOrder int
		_ = tx.Model(&FlightRequestRouteRow{}).
			Where("flight_request_id = ?", draft.ID).
			Select("COALESCE(MAX(segment_order), 0)").
			Scan(&maxOrder).Error

		row := FlightRequestRouteRow{
			FlightRequestID:  draft.ID,
			RouteID:          routeID,
			Quantity:         1,
			SegmentOrder:     maxOrder + 1,
			IsPrimary:        maxOrder == 0,
			SegmentDryMassKg: &draft.DryMassKg,
			SegmentIspSec:    &draft.IspSeconds,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "flight_request_id"}, {Name: "route_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"quantity": gorm.Expr("flight_request_routes.quantity + 1"),
			}),
		}).Create(&row).Error; err != nil {
			return err
		}

		return tx.Preload("Items.Route").First(&out, draft.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SoftDeleteDraft — логическое удаление заявки через SQL UPDATE (по заданию: без ORM).
func (r *Repository) SoftDeleteDraft(userID, missionID int) error {
	return r.db.Exec(
		"UPDATE flight_requests SET status = 'deleted' WHERE id = ? AND created_by = ? AND status = 'draft'",
		missionID, userID,
	).Error
}

// UpdateSegmentParams обновляет параметры сегмента и (опционально) сохраняет расчёты.
func (r *Repository) UpdateSegmentParams(userID, missionID, routeID int, mass, isp float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var fr FlightRequest
		if err := tx.Where("id = ? AND created_by = ?", missionID, userID).First(&fr).Error; err != nil {
			return err
		}

		var row FlightRequestRouteRow
		if err := tx.Preload("Route").
			Where("flight_request_id = ? AND route_id = ?", missionID, routeID).
			First(&row).Error; err != nil {
			return err
		}

		// пересчёт по текущим формулам
		dv := CalculateDeltaVms(row.Route.FromOrbitKm, row.Route.ToOrbitKm)
		fuel := CalculatePropellantKg(mass, dv, isp)

		return tx.Model(&row).Updates(map[string]interface{}{
			"segment_dry_mass_kg": mass,
			"segment_isp_sec":     isp,
			"delta_v_kms":         dv,
			"fuel_mass_kg":        fuel,
		}).Error
	})
}

// RemoveRouteFromDraft удаляет один маршрут из черновика.
func (r *Repository) RemoveRouteFromDraft(userID, missionID, routeID int) error {
	return r.db.Exec(`
DELETE FROM flight_request_routes frt
USING flight_requests fr
WHERE fr.id = frt.flight_request_id
  AND fr.created_by = ?
  AND fr.status = 'draft'
  AND frt.flight_request_id = ?
  AND frt.route_id = ?`,
		userID, missionID, routeID,
	).Error
}

// UpdateSegmentMM обновляет поля м-м строки (quantity, order, параметры сегмента)
// и, при изменении массы/удельного импульса, пересчитывает dv/топливо.
func (r *Repository) UpdateSegmentMM(userID, missionID, routeID int, quantity, segmentOrder *int, mass, isp *float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var fr FlightRequest
		if err := tx.Where("id = ? AND created_by = ? AND status = 'draft'", missionID, userID).First(&fr).Error; err != nil {
			return err
		}

		var row FlightRequestRouteRow
		if err := tx.Preload("Route").
			Where("flight_request_id = ? AND route_id = ?", missionID, routeID).
			First(&row).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{}
		if quantity != nil {
			updates["quantity"] = *quantity
		}
		if segmentOrder != nil {
			updates["segment_order"] = *segmentOrder
		}

		var dv, fuel float64
		if mass != nil {
			updates["segment_dry_mass_kg"] = *mass
		}
		if isp != nil {
			updates["segment_isp_sec"] = *isp
		}
		if mass != nil || isp != nil {
			m := fr.DryMassKg
			if mass != nil {
				m = *mass
			} else if row.SegmentDryMassKg != nil {
				m = *row.SegmentDryMassKg
			}
			i := fr.IspSeconds
			if isp != nil {
				i = *isp
			} else if row.SegmentIspSec != nil {
				i = *row.SegmentIspSec
			}

			dv = CalculateDeltaVms(row.Route.FromOrbitKm, row.Route.ToOrbitKm)
			fuel = CalculatePropellantKg(m, dv, i)
			updates["delta_v_kms"] = dv
			updates["fuel_mass_kg"] = fuel
		}

		if len(updates) == 0 {
			return nil
		}

		return tx.Model(&row).Updates(updates).Error
	})
}

// RequestListItem — DTO для списка заявок с логинами и итогами.
type RequestListItem struct {
	ID                 int        `json:"id"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	FormedAt           *time.Time `json:"formed_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	CreatorLogin       string     `json:"creator_login"`
	ModeratorLogin     *string    `json:"moderator_login"`
	TotalFuelMassKg    *float64   `json:"total_fuel_mass_kg"`
	SegmentsWithResult int64      `json:"segments_with_result"`
}

// ListRequests возвращает список заявок с фильтрацией по статусу и диапазону даты формирования.
func (r *Repository) ListRequests(status string, formedFrom, formedTo *time.Time) ([]RequestListItem, error) {
	var items []RequestListItem

	tx := r.db.Table("flight_requests fr").
		Select(`fr.id,
		        fr.status,
		        fr.created_at,
		        fr.formed_at,
		        fr.completed_at,
		        uc.username AS creator_login,
		        um.username AS moderator_login,
		        fr.total_fuel_mass_kg,
		        COALESCE((
		          SELECT COUNT(*)
		          FROM flight_request_routes mm
		          WHERE mm.flight_request_id = fr.id
		            AND mm.fuel_mass_kg IS NOT NULL
		        ), 0) AS segments_with_result`).
		Joins("JOIN users uc ON uc.id = fr.created_by").
		Joins("LEFT JOIN users um ON um.id = fr.moderated_by").
		Where("fr.status NOT IN ('deleted', 'draft')")

	if status != "" {
		tx = tx.Where("fr.status = ?", status)
	}
	if formedFrom != nil {
		tx = tx.Where("fr.formed_at >= ?", *formedFrom)
	}
	if formedTo != nil {
		tx = tx.Where("fr.formed_at <= ?", *formedTo)
	}

	if err := tx.Order("fr.formed_at DESC, fr.id DESC").Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetRequestWithItems возвращает одну заявку с услугами для API.
func (r *Repository) GetRequestWithItems(userID, id int) (*FlightRequest, error) {
	var fr FlightRequest
	err := r.db.Preload("Items.Route").
		Where("id = ? AND created_by = ? AND status <> 'deleted'", id, userID).
		First(&fr).Error
	if err != nil {
		return nil, err
	}
	return &fr, nil
}

// UpdateRequestFields изменяет только «тематические» поля заявки.
func (r *Repository) UpdateRequestFields(userID, id int, dryMassKg, ispSeconds *float64) error {
	updates := map[string]interface{}{}
	if dryMassKg != nil {
		updates["spacecraft_dry_mass_kg"] = *dryMassKg
	}
	if ispSeconds != nil {
		updates["engine_isp_sec"] = *ispSeconds
	}
	if len(updates) == 0 {
		return nil
	}

	return r.db.Model(&FlightRequest{}).
		Where("id = ? AND created_by = ? AND status = 'draft'", id, userID).
		Updates(updates).Error
}

// FormRequest устанавливает статус 'formed', дату формирования и пересчитывает итоговое топливо.
func (r *Repository) FormRequest(userID, id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var fr FlightRequest
		if err := tx.Preload("Items.Route").
			Where("id = ? AND created_by = ? AND status = 'draft'", id, userID).
			First(&fr).Error; err != nil {
			return err
		}

		if len(fr.Items) == 0 {
			return fmt.Errorf("заявка пуста, добавить хотя бы один маршрут")
		}

		if fr.DryMassKg <= 0 || fr.IspSeconds <= 0 {
			return fmt.Errorf("обязательные поля заявки не заполнены корректно")
		}

		var totalFuel float64
		for _, it := range fr.Items {
			mass := fr.DryMassKg
			if it.SegmentDryMassKg != nil && *it.SegmentDryMassKg > 0 {
				mass = *it.SegmentDryMassKg
			}
			isp := fr.IspSeconds
			if it.SegmentIspSec != nil && *it.SegmentIspSec > 0 {
				isp = *it.SegmentIspSec
			}

			dv := CalculateDeltaVms(it.Route.FromOrbitKm, it.Route.ToOrbitKm)
			fuel := CalculatePropellantKg(mass, dv, isp)
			totalFuel += fuel

			if err := tx.Model(&FlightRequestRouteRow{}).
				Where("flight_request_id = ? AND route_id = ?", fr.ID, it.RouteID).
				Updates(map[string]interface{}{
					"segment_dry_mass_kg": mass,
					"segment_isp_sec":     isp,
					"delta_v_kms":         dv,
					"fuel_mass_kg":        fuel,
				}).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		return tx.Model(&fr).Updates(map[string]interface{}{
			"status":             "formed",
			"formed_at":          now,
			"total_fuel_mass_kg": totalFuel,
		}).Error
	})
}

// ModerateRequest завершает или отклоняет сформированную заявку.
func (r *Repository) ModerateRequest(id, moderatorID int, action string) error {
	newStatus := ""
	switch action {
	case "complete":
		newStatus = "completed"
	case "reject":
		newStatus = "rejected"
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	var fr FlightRequest
	if err := r.db.Where("id = ?", id).First(&fr).Error; err != nil {
		return err
	}
	if fr.Status != "formed" {
		return fmt.Errorf("завершить или отклонить можно только сформированную заявку")
	}

	now := time.Now()
	result := r.db.Model(&FlightRequest{}).
		Where("id = ? AND status = 'formed'", id).
		Updates(map[string]interface{}{
			"status":       newStatus,
			"completed_at": now,
			"moderated_by": moderatorID,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("заявка не обновлена")
	}
	return nil
}

// SoftDeleteRequest помечает заявку удалённой (только черновик создателя).
func (r *Repository) SoftDeleteRequest(userID, id int) error {
	return r.db.Exec(
		"UPDATE flight_requests SET status = 'deleted' WHERE id = ? AND created_by = ? AND status = 'draft'",
		id, userID,
	).Error
}

// CreateUser регистрирует нового пользователя.
func (r *Repository) CreateUser(username, passwordHash string, isModerator bool) (*User, error) {
	u := &User{
		Username:     username,
		PasswordHash: passwordHash,
		IsModerator:  isModerator,
	}
	if err := r.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// CalculateDeltaVms реализует формулу Гомана по радиусам орбит (км).
func CalculateDeltaVms(rEarthKm, rTargetKm float64) float64 {
	if rEarthKm <= 0 || rTargetKm <= 0 {
		return 0
	}

	// Переводим км -> м
	r3 := rEarthKm * 1000
	r := rTargetKm * 1000

	a := (r3 + r) / 2
	v3 := math.Sqrt(muSun / r3)
	vp := math.Sqrt(muSun * (2/r3 - 1/a))
	dv1 := math.Abs(vp - v3)
	vc := math.Sqrt(muSun / r)
	va := math.Sqrt(muSun * (2/r - 1/a))
	dv2 := math.Abs(vc - va)
	return dv1 + dv2
}

func CalculatePropellantKg(dryMassKg, deltaVms, ispSeconds float64) float64 {
	if dryMassKg <= 0 || deltaVms <= 0 || ispSeconds <= 0 {
		return 0
	}
	return dryMassKg * (math.Exp(deltaVms/(ispSeconds*g0)) - 1)
}

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

// ToMissionProfile — собирает DTO для шаблонов с пересчётом dv/топлива/энергии из БД-полей.
func ToMissionProfile(fr *FlightRequest) MissionProfile {
	mp := MissionProfile{
		ID:          fr.ID,
		Title:       "Заявка на расчёт межпланетного перелёта",
		Description: "Параметры аппарата и выбранные маршруты для оценки Δv, массы топлива и энергии.",
		DryMassKg:   fr.DryMassKg,
		IspSeconds:  fr.IspSeconds,
	}

	for _, it := range fr.Items {
		route := it.Route
		// для каждой строки свои параметры, если заданы; иначе берём из заявки
		mass := fr.DryMassKg
		if it.SegmentDryMassKg != nil {
			mass = *it.SegmentDryMassKg
		}
		isp := fr.IspSeconds
		if it.SegmentIspSec != nil {
			isp = *it.SegmentIspSec
		}

		dv := CalculateDeltaVms(route.FromOrbitKm, route.ToOrbitKm)
		fuel := CalculatePropellantKg(mass, dv, isp)
		energy := CalculateEnergyJ(mass, fuel, dv)

		mp.Routes = append(mp.Routes, MissionRoute{
			Route:            route,
			DeltaVms:         dv,
			PropellantKg:     fuel,
			EnergyJ:          energy,
			SegmentDryMassKg: mass,
			SegmentIspSec:    isp,
		})
	}
	mp.RouteCount = len(mp.Routes)
	return mp
}
