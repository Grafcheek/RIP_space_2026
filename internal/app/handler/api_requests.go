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
// @Summary Иконка корзины (черновик)
// @Tags interplanetaryflightrequests
// @Produce json
// @Success 200 {object} map[string]interface{} "id, count"
// @Failure 500 {object} map[string]string
// @Router /interplanetaryflightrequests/cart-icon [get]
func (h *Handler) APIGetInterplanetaryFlightRequestCartIcon(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil || userID == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"id":    nil,
			"count": 0,
		})
		return
	}

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
// @Summary Список заявок (создатель — свои; модератор — все)
// @Tags interplanetaryflightrequests
// @Produce json
// @Param status query string false "Статус"
// @Param from query string false "formed_at с (RFC3339)"
// @Param to query string false "formed_at по (RFC3339)"
// @Success 200 {object} map[string]interface{} "items"
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests [get]
func (h *Handler) APIListInterplanetaryFlightRequests(ctx *gin.Context) {
	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	isMod := isModeratorFromContext(ctx)

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

	items, err := h.Repository.ListRequests(uid, isMod, status, formedFrom, formedTo)
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
// @Summary Детальная заявка с расчётами Δv / топливо / энергия
// @Tags interplanetaryflightrequests
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} InterplanetaryFlightRequestDetail
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id} [get]
func (h *Handler) APIGetInterplanetaryFlightRequest(ctx *gin.Context) {
	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	isMod := isModeratorFromContext(ctx)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	fr, err := h.Repository.GetRequestWithItems(uid, id, isMod)
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

// UpdateInterplanetaryFlightRequestBody PATCH полей заявки (swagger).
type UpdateInterplanetaryFlightRequestBody struct {
	DryMassKg  *float64 `json:"spacecraft_dry_mass_kg"`
	IspSeconds *float64 `json:"engine_isp_sec"`
}

// APIUpdateInterplanetaryFlightRequest — тематические поля заявки на межпланетный перелёт.
// @Summary Редактирование черновика заявки
// @Tags interplanetaryflightrequests
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param body body UpdateInterplanetaryFlightRequestBody true "Масса сухого аппарата, Isp"
// @Success 204 "OK"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id} [put]
func (h *Handler) APIUpdateInterplanetaryFlightRequest(ctx *gin.Context) {
	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var payload UpdateInterplanetaryFlightRequestBody
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.UpdateRequestFields(uid, id, payload.DryMassKg, payload.IspSeconds); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// APIFormInterplanetaryFlightRequest — формирование заявки на межпланетный перелёт создателем.
// @Summary Сформировать заявку (draft → formed)
// @Tags interplanetaryflightrequests
// @Produce json
// @Param id path int true "ID заявки"
// @Success 200 {object} InterplanetaryFlightRequestDetail
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id}/form [put]
func (h *Handler) APIFormInterplanetaryFlightRequest(ctx *gin.Context) {
	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.FormRequest(uid, id); err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	fr, err := h.Repository.GetRequestWithItems(uid, id, false)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	// 200 + тело: в Postman видно результат формирования без отдельного GET (преподавательский сценарий).
	ctx.JSON(http.StatusOK, buildInterplanetaryFlightRequestDetail(fr))
}

// ModerateInterplanetaryFlightRequestBody действие модератора (swagger).
type ModerateInterplanetaryFlightRequestBody struct {
	Action string `json:"action" binding:"required"` // "complete" | "reject"
}

// APIModerateInterplanetaryFlightRequest — завершение или отклонение сформированной заявки модератором.
// @Summary Модерация заявки (только модератор)
// @Tags interplanetaryflightrequests
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param body body ModerateInterplanetaryFlightRequestBody true "complete | reject"
// @Success 204 "OK"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id}/moderate [put]
func (h *Handler) APIModerateInterplanetaryFlightRequest(ctx *gin.Context) {
	moderatorID, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var payload ModerateInterplanetaryFlightRequestBody
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.ModerateRequest(id, moderatorID, payload.Action); err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// APIDeleteInterplanetaryFlightRequest — логическое удаление черновика заявки на межпланетный перелёт.
// @Summary Удалить черновик заявки
// @Tags interplanetaryflightrequests
// @Produce json
// @Param id path int true "ID заявки"
// @Success 204 "OK"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightrequests/{id} [delete]
func (h *Handler) APIDeleteInterplanetaryFlightRequest(ctx *gin.Context) {
	uid, err := getUserID(ctx)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.SoftDeleteRequest(uid, id); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Status(http.StatusNoContent)
}
