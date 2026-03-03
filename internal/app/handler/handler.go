package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

	missions, err := h.Repository.GetMissionProfiles()
	if err != nil {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"routes":   routes,
		"query":    searchQuery,
		"missions": missions,
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

	missionRoute, err := h.Repository.GetMissionRouteForRoute(id)
	hasMissionData := err == nil && missionRoute != nil

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

	mission, err := h.Repository.GetMissionProfile(id)
	if err != nil {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "system_load.html", gin.H{
		"mission": mission,
	})
}
