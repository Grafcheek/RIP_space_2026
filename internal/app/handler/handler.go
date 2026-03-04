package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"web_backend/internal/app/repository"
	"gorm.io/gorm"
)

const demoUserID = 1

// Handler содержит зависимости HTTP-обработчиков.
type Handler struct {
	Repository *repository.Repository
}

// NewHandler создаёт новый Handler с переданным репозиторием.
func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

// GetRoutes — главная страница: каталог межпланетных маршрутов + иконка заявки (миссии).
func (h *Handler) GetRoutes(ctx *gin.Context) {
	searchQuery := ctx.Query("query")

	routes, err := h.Repository.SearchRoutes(searchQuery)
	if err != nil {
		logrus.Error(err)
	}

	var (
		hasDraft bool
		draftMP  repository.MissionProfile
	)
	if fr, err := h.Repository.GetDraft(demoUserID); err == nil {
		hasDraft = true
		draftMP = repository.ToMissionProfile(fr)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"routes":    routes,
		"query":     searchQuery,
		"hasDraft":  hasDraft,
		"draft":     draftMP,
		"draftID":   draftMP.ID,
		"draftSize": draftMP.RouteCount,
	})
}

// GetRoute — детальная страница маршрута (Земля → планета) с расчётами.
func (h *Handler) GetRoute(ctx *gin.Context) {
	idStr := ctx.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	route, err := h.Repository.GetRoute(id)
	if err != nil {
		logrus.Error(err)
	}

	var (
		hasMissionData bool
		missionRoute   *repository.MissionRoute
	)
	if fr, err := h.Repository.GetDraft(demoUserID); err == nil {
		mp := repository.ToMissionProfile(fr)
		for i := range mp.Routes {
			if mp.Routes[i].Route.ID == route.ID {
				hasMissionData = true
				missionRoute = &mp.Routes[i]
				break
			}
		}
	} else if err != nil && err != gorm.ErrRecordNotFound {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "strategy.html", gin.H{
		"route":          route,
		"missionRoute":   missionRoute,
		"hasMissionData": hasMissionData,
	})
}

// GetMission — страница заявки: профиль миссии и таблица маршрутов с Δv, топливом и энергией.
func (h *Handler) GetMission(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	fr, err := h.Repository.GetMission(demoUserID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Status(http.StatusNotFound)
			return
		}
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	mission := repository.ToMissionProfile(fr)
	mission.CanDelete = fr.Status == "draft"

	ctx.HTML(http.StatusOK, "system_load.html", gin.H{
		"mission": mission,
	})
}

// PostAddToDraft — добавление услуги в текущую заявку через ORM.
func (h *Handler) PostAddToDraft(ctx *gin.Context) {
	idStr := ctx.Param("id")
	routeID, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if _, err := h.Repository.AddRouteToDraft(demoUserID, routeID); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}

// PostDeleteDraft — логическое удаление заявки через SQL UPDATE (без ORM).
func (h *Handler) PostDeleteDraft(ctx *gin.Context) {
	idStr := ctx.Param("id")
	missionID, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.SoftDeleteDraft(demoUserID, missionID); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/")
}

// PostRecalcSegment — пересчёт одного сегмента заявки с новыми параметрами.
func (h *Handler) PostRecalcSegment(ctx *gin.Context) {
	missionIDStr := ctx.Param("id")
	routeIDStr := ctx.Param("routeId")

	missionID, err := strconv.Atoi(missionIDStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	routeID, err := strconv.Atoi(routeIDStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	massStr := ctx.PostForm("dry_mass_kg")
	ispStr := ctx.PostForm("isp_sec")

	mass, err := strconv.ParseFloat(massStr, 64)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	isp, err := strconv.ParseFloat(ispStr, 64)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.UpdateSegmentParams(demoUserID, missionID, routeID, mass, isp); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/missions/"+missionIDStr)
}

// PostDeleteSegment — удаление одного маршрута из заявки.
func (h *Handler) PostDeleteSegment(ctx *gin.Context) {
	missionIDStr := ctx.Param("id")
	routeIDStr := ctx.Param("routeId")

	missionID, err := strconv.Atoi(missionIDStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	routeID, err := strconv.Atoi(routeIDStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.RemoveRouteFromDraft(demoUserID, missionID, routeID); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/missions/"+missionIDStr)
}
