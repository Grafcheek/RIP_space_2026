package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"web_backend/internal/app/repository"
)

// RegisterUserPayload тело запроса регистрации (swagger).
type RegisterUserPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginUserPayload тело запроса входа (swagger).
type LoginUserPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// APIRegisterInterplanetaryFlightsUser регистрирует пользователя сервиса межпланетных перелётов.
// @Summary Регистрация пользователя
// @Tags interplanetaryflightusers
// @Accept json
// @Produce json
// @Param body body RegisterUserPayload true "Логин и пароль"
// @Success 201 {object} map[string]interface{} "id, username"
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string "Логин занят"
// @Failure 500 {object} map[string]string
// @Router /interplanetaryflightusers/register [post]
func (h *Handler) APIRegisterInterplanetaryFlightsUser(ctx *gin.Context) {
	var payload RegisterUserPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	user, err := h.Repository.CreateUser(payload.Username, payload.Password, false)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			h.apiError(ctx, http.StatusConflict, err)
			return
		}
		h.apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// APILoginInterplanetaryFlightsUser выдаёт JWT после проверки логина и пароля.
// @Summary Вход, получение JWT
// @Tags interplanetaryflightusers
// @Accept json
// @Produce json
// @Param body body LoginUserPayload true "Логин и пароль"
// @Success 200 {object} map[string]string "token"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /interplanetaryflightusers/login [post]
func (h *Handler) APILoginInterplanetaryFlightsUser(ctx *gin.Context) {
	var payload LoginUserPayload
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	token, err := h.Repository.SignIn(payload.Username, payload.Password)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCredentials) {
			h.apiError(ctx, http.StatusUnauthorized, err)
			return
		}
		h.apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": token})
}

// APILogoutInterplanetaryFlightsUser помещает текущий JWT в Redis blacklist.
// @Summary Выход (blacklist токена)
// @Tags interplanetaryflightusers
// @Produce json
// @Success 204 "Токен отозван"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /interplanetaryflightusers/logout [post]
func (h *Handler) APILogoutInterplanetaryFlightsUser(ctx *gin.Context) {
	tokenString := extractTokenFromHeader(ctx.Request)
	if tokenString == "" {
		h.apiError(ctx, http.StatusUnauthorized, errors.New("no token provided"))
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtKeyBytes(), nil
	})
	if err != nil || token == nil {
		h.apiError(ctx, http.StatusUnauthorized, err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		h.apiError(ctx, http.StatusBadRequest, errors.New("invalid token claims"))
		return
	}

	ttl, err := tokenTTLFromClaims(claims)
	if err != nil {
		ctx.Status(http.StatusNoContent)
		return
	}

	userID, _ := claims["user_id"].(string)
	if userID == "" {
		userID = "unknown"
	}

	if err := h.Repository.AddTokenToBlacklist(context.Background(), tokenString, ttl, userID); err != nil {
		h.apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
