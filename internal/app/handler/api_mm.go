package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"web_backend/internal/app/repository"
)

// APIDeleteInterplanetaryFlightInRequest — удаление межпланетного перелёта из строки м-м заявки.
// @Summary Удалить перелёт из заявки
// @Tags interplanetaryflightrequestitems
// @Produce json
// @Param id path int true "ID заявки"
// @Param routeId path int true "ID маршрута"
// @Success 204 "OK"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id}/items/{routeId} [delete]
func (h *Handler) APIDeleteInterplanetaryFlightInRequest(ctx *gin.Context) {
	missionID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	routeID, err := strconv.Atoi(ctx.Param("routeId"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	if err := h.Repository.RemoveRouteFromDraft(uid, missionID, routeID); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UpdateInterplanetaryFlightMMBody тело PUT строки м-м (swagger).
type UpdateInterplanetaryFlightMMBody struct {
	Quantity     *int     `json:"quantity"`
	SegmentOrder *int     `json:"segment_order"`
	MassKg       *float64 `json:"segment_dry_mass_kg"`
	IspSec       *float64 `json:"segment_isp_sec"`
}

// APIUpdateInterplanetaryFlightInRequest — поля м-м (количество, порядок, параметры сегмента перелёта).
// @Summary Изменить строку м-м (пересчёт Δv/топливо при смене массы/Isp)
// @Tags interplanetaryflightrequestitems
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param routeId path int true "ID маршрута"
// @Param body body UpdateInterplanetaryFlightMMBody true "Поля"
// @Success 204 "OK"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id}/items/{routeId} [put]
func (h *Handler) APIUpdateInterplanetaryFlightInRequest(ctx *gin.Context) {
	missionID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	routeID, err := strconv.Atoi(ctx.Param("routeId"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var req UpdateInterplanetaryFlightMMBody
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	if err := h.Repository.UpdateSegmentMM(uid, missionID, routeID, req.Quantity, req.SegmentOrder, req.MassKg, req.IspSec); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddInterplanetaryFlightToDraftBody добавление услуги в черновик (swagger).
type AddInterplanetaryFlightToDraftBody struct {
	RouteID int `json:"route_id" binding:"required"`
}

// APIAddInterplanetaryFlightToDraftRequest — добавление межпланетного перелёта в черновик заявки.
// @Summary Добавить межпланетный перелёт в черновик (created_by из JWT)
// @Tags interplanetaryflightrequestitems
// @Accept json
// @Produce json
// @Param body body AddInterplanetaryFlightToDraftBody true "route_id"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/draft/items [post]
func (h *Handler) APIAddInterplanetaryFlightToDraftRequest(ctx *gin.Context) {
	var req AddInterplanetaryFlightToDraftBody
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	fr, err := h.Repository.AddRouteToDraft(uid, req.RouteID)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	mp := repository.ToMissionProfile(fr)
	mp.CanDelete = fr.Status == "draft"
	ctx.JSON(http.StatusOK, mp)
}
