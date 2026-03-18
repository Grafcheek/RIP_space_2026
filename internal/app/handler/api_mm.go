package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"web_backend/internal/app/repository"
)

// APIDeleteRequestItem удаляет услугу из заявки-черновика.
// DELETE /api/requests/:id/items/:routeId
func (h *Handler) APIDeleteRequestItem(ctx *gin.Context) {
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

// APIUpdateRequestItem изменяет параметры м-м строки (количество, порядок, параметры сегмента).
// PUT /api/requests/:id/items/:routeId
func (h *Handler) APIUpdateRequestItem(ctx *gin.Context) {
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

// APIAddToDraft добавляет услугу в заявку-черновик, при необходимости создавая её.
// POST /api/requests/draft/items
func (h *Handler) APIAddToDraft(ctx *gin.Context) {
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

	ctx.JSON(http.StatusOK, repository.ToMissionProfile(fr))
}
