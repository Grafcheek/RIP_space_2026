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

// APIRegisterUser регистрирует нового пользователя.
// POST /api/users/register
func (h *Handler) APIRegisterUser(ctx *gin.Context) {
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

// APILoginUser — заглушка для аутентификации (будет реализована в лаб. 4).
// POST /api/users/login
func (h *Handler) APILoginUser(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "login stub: здесь в лаб.4 будет настоящая аутентификация",
	})
}

// APILogoutUser — заглушка для деавторизации (будет реализована в лаб. 4).
// POST /api/users/logout
func (h *Handler) APILogoutUser(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "logout stub: здесь в лаб.4 будет реальный выход из системы",
	})
}
