package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type registerUserPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// APIRegisterInterplanetaryFlightsUser — регистрация пользователя сервиса межпланетных перелётов.
// POST /api/interplanetaryflightusers/register
func (h *Handler) APIRegisterInterplanetaryFlightsUser(ctx *gin.Context) {
	var payload registerUserPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Для лабораторной пароль храним как есть (хеширование будет в следующей работе).
	user, err := h.Repository.CreateUser(payload.Username, payload.Password, false)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// APILoginInterplanetaryFlightsUser — заглушка входа (лаб. 4).
// POST /api/users/login
func (h *Handler) APILoginInterplanetaryFlightsUser(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "login stub: здесь в лаб.4 будет настоящая аутентификация",
	})
}

// APILogoutInterplanetaryFlightsUser — заглушка выхода (лаб. 4).
// POST /api/interplanetaryflightusers/logout
func (h *Handler) APILogoutInterplanetaryFlightsUser(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "logout stub: здесь в лаб.4 будет реальный выход из системы",
	})
}
