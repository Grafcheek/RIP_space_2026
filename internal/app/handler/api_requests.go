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

// APIGetInterplanetaryFlightRequestCartIcon — иконка корзины (черновик заявки на межпланетный перелёт).
// GET /api/interplanetaryflightrequests/cart-icon
func (h *Handler) APIGetInterplanetaryFlightRequestCartIcon(ctx *gin.Context) {
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

// APIListInterplanetaryFlightRequests — список заявок на межпланетный перелёт (статус, даты формирования).
// GET /api/interplanetaryflightrequests?status=&from=&to=
func (h *Handler) APIListInterplanetaryFlightRequests(ctx *gin.Context) {
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

// APIGetInterplanetaryFlightRequest — одна заявка на межпланетный перелёт с перелётами в составе.
// GET /api/interplanetaryflightrequests/:id
func (h *Handler) APIGetInterplanetaryFlightRequest(ctx *gin.Context) {
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

	ctx.JSON(http.StatusOK, buildInterplanetaryFlightRequestDetail(fr))
}

type updateRequestPayload struct {
	DryMassKg  *float64 `json:"spacecraft_dry_mass_kg"`
	IspSeconds *float64 `json:"engine_isp_sec"`
}

// APIUpdateInterplanetaryFlightRequest — тематические поля заявки на межпланетный перелёт.
// PUT /api/interplanetaryflightrequests/:id
func (h *Handler) APIUpdateInterplanetaryFlightRequest(ctx *gin.Context) {
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

// APIFormInterplanetaryFlightRequest — формирование заявки на межпланетный перелёт создателем.
// PUT /api/interplanetaryflightrequests/:id/form
func (h *Handler) APIFormInterplanetaryFlightRequest(ctx *gin.Context) {
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

	fr, err := h.Repository.GetRequestWithItems(CurrentUserID(), id)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	// 200 + тело: в Postman видно результат формирования без отдельного GET (преподавательский сценарий).
	ctx.JSON(http.StatusOK, buildInterplanetaryFlightRequestDetail(fr))
}

type moderateRequestPayload struct {
	Action string `json:"action" binding:"required"` // "complete" | "reject"
}

// APIModerateInterplanetaryFlightRequest — завершение или отклонение сформированной заявки модератором.
// PUT /api/interplanetaryflightrequests/:id/moderate
func (h *Handler) APIModerateInterplanetaryFlightRequest(ctx *gin.Context) {
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

// APIDeleteInterplanetaryFlightRequest — логическое удаление черновика заявки на межпланетный перелёт.
// DELETE /api/interplanetaryflightrequests/:id
func (h *Handler) APIDeleteInterplanetaryFlightRequest(ctx *gin.Context) {
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
