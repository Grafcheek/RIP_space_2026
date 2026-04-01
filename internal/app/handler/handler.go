package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"web_backend/internal/app/repository"
)

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

	var draftMP repository.MissionProfile
	if fr, err := h.Repository.GetDraft(CurrentUserID()); err == nil {
		draftMP = repository.ToMissionProfile(fr)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		logrus.Error(err)
	}

	basketActive := draftMP.ID != 0 && draftMP.RouteCount > 0

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"routes":       routes,
		"query":        searchQuery,
		"draftID":      draftMP.ID,
		"draftSize":    draftMP.RouteCount,
		"basketActive": basketActive,
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

	searchQuery := ctx.Query("query")

	var (
		hasMissionData bool
		missionRoute   *repository.MissionRoute
		draftMP        repository.MissionProfile
	)
	if fr, err := h.Repository.GetDraft(CurrentUserID()); err == nil {
		draftMP = repository.ToMissionProfile(fr)
		mp := draftMP
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

	basketActive := draftMP.ID != 0 && draftMP.RouteCount > 0

	ctx.HTML(http.StatusOK, "strategy.html", gin.H{
		"route":          route,
		"missionRoute":   missionRoute,
		"hasMissionData": hasMissionData,
		"query":          searchQuery,
		"draftID":        draftMP.ID,
		"draftSize":      draftMP.RouteCount,
		"basketActive":   basketActive,
	})
}

// GetMission — страница заявки: профиль миссии и таблица маршрутов с Δv, топливом и энергией.
func (h *Handler) GetMission(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	fr, err := h.Repository.GetMission(CurrentUserID(), id)
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

	var draftMP repository.MissionProfile
	if dfr, err := h.Repository.GetDraft(CurrentUserID()); err == nil {
		draftMP = repository.ToMissionProfile(dfr)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		logrus.Error(err)
	}
	basketActive := draftMP.ID != 0 && draftMP.RouteCount > 0

	ctx.HTML(http.StatusOK, "system_load.html", gin.H{
		"mission":      mission,
		"query":        "",
		"draftID":      draftMP.ID,
		"draftSize":    draftMP.RouteCount,
		"basketActive": basketActive,
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

	if _, err := h.Repository.AddRouteToDraft(CurrentUserID(), routeID); err != nil {
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

	if err := h.Repository.SoftDeleteDraft(CurrentUserID(), missionID); err != nil {
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

	if err := h.Repository.UpdateSegmentParams(CurrentUserID(), missionID, routeID, mass, isp); err != nil {
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

	if err := h.Repository.RemoveRouteFromDraft(CurrentUserID(), missionID, routeID); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/missions/"+missionIDStr)
}
