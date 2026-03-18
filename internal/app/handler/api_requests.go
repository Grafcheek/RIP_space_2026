package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"web_backend/internal/app/repository"
)

// APICartIcon возвращает информацию о корзине (текущей заявке-черновике).
// GET /api/requests/cart-icon
func (h *Handler) APICartIcon(ctx *gin.Context) {
	userID := CurrentUserID()

	fr, err := h.Repository.GetDraft(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusOK, gin.H{
				"id":    nil,
				"count": 0,
			})
			return
		}
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	mp := repository.ToMissionProfile(fr)
	ctx.JSON(http.StatusOK, gin.H{
		"id":    mp.ID,
		"count": mp.RouteCount,
	})
}

// APIGetRequests возвращает список заявок с фильтрацией по статусу и диапазону даты формирования.
// GET /api/requests?status=&from=&to=
func (h *Handler) APIGetRequests(ctx *gin.Context) {
	status := ctx.Query("status")

	var formedFrom, formedTo *time.Time
	if v := ctx.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			formedFrom = &t
		}
	}
	if v := ctx.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			formedTo = &t
		}
	}

	items, err := h.Repository.ListRequests(status, formedFrom, formedTo)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items": items,
	})
}

// APIGetRequest возвращает одну заявку с услугами.
// GET /api/requests/:id
func (h *Handler) APIGetRequest(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	fr, err := h.Repository.GetRequestWithItems(CurrentUserID(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.Status(http.StatusNotFound)
			return
		}
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, repository.ToMissionProfile(fr))
}

type updateRequestPayload struct {
	DryMassKg  *float64 `json:"spacecraft_dry_mass_kg"`
	IspSeconds *float64 `json:"engine_isp_sec"`
}

// APIUpdateRequest изменяет тематические поля заявки.
// PUT /api/requests/:id
func (h *Handler) APIUpdateRequest(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var payload updateRequestPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.UpdateRequestFields(CurrentUserID(), id, payload.DryMassKg, payload.IspSeconds); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// APIFormRequest выполняет формирование заявки создателем.
// PUT /api/requests/:id/form
func (h *Handler) APIFormRequest(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.FormRequest(CurrentUserID(), id); err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

type moderateRequestPayload struct {
	Action string `json:"action" binding:"required"` // "complete" | "reject"
}

// APIModerateRequest завершает или отклоняет сформированную заявку модератором.
// PUT /api/requests/:id/moderate
func (h *Handler) APIModerateRequest(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var payload moderateRequestPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.ModerateRequest(id, CurrentModeratorID(), payload.Action); err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// APIDeleteRequest логически удаляет заявку-черновик.
// DELETE /api/requests/:id
func (h *Handler) APIDeleteRequest(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.SoftDeleteRequest(CurrentUserID(), id); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
