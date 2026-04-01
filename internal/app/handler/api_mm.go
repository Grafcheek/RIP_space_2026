package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"web_backend/internal/app/repository"
)

// APIDeleteInterplanetaryFlightInRequest — удаление межпланетного перелёта из строки м-м заявки.
// DELETE /api/interplanetaryflightrequests/:id/items/:routeId
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

	if err := h.Repository.RemoveRouteFromDraft(CurrentUserID(), missionID, routeID); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

type updateMMRequest struct {
	Quantity     *int     `json:"quantity"`
	SegmentOrder *int     `json:"segment_order"`
	MassKg       *float64 `json:"segment_dry_mass_kg"`
	IspSec       *float64 `json:"segment_isp_sec"`
}

// APIUpdateInterplanetaryFlightInRequest — поля м-м (количество, порядок, параметры сегмента перелёта).
// PUT /api/requests/:id/items/:routeId
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

	var req updateMMRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.UpdateSegmentMM(CurrentUserID(), missionID, routeID, req.Quantity, req.SegmentOrder, req.MassKg, req.IspSec); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

type addToDraftRequest struct {
	RouteID int `json:"route_id" binding:"required"`
}

// APIAddInterplanetaryFlightToDraftRequest — добавление межпланетного перелёта в черновик заявки.
// POST /api/interplanetaryflightrequests/draft/items
func (h *Handler) APIAddInterplanetaryFlightToDraftRequest(ctx *gin.Context) {
	var req addToDraftRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	fr, err := h.Repository.AddRouteToDraft(CurrentUserID(), req.RouteID)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	mp := repository.ToMissionProfile(fr)
	mp.CanDelete = fr.Status == "draft"
	ctx.JSON(http.StatusOK, mp)
}
