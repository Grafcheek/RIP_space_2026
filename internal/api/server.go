package api

import (
	"log"

	"web_backend/internal/app/handler"
	"web_backend/internal/app/repository"
	"web_backend/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// StartServer инициализирует репозиторий, обработчики и запускает HTTP-сервер.
func StartServer() {
	log.Println("Starting server")

	gormDB, err := db.OpenPostgres(db.ConfigFromEnv())
	if err != nil {
		logrus.Fatalf("Ошибка подключения к Postgres: %v", err)
	}

	repo, err := repository.NewRepository(gormDB)
	if err != nil {
		logrus.Fatalf("Ошибка инициализации репозитория: %v", err)
	}

	h := handler.NewHandler(repo)

	r := gin.Default()

	// HTML-приложение (лаб. 1-2)
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/", h.GetRoutes)
	r.GET("/routes/:id", h.GetRoute)
	r.GET("/missions/:id", h.GetMission)
	r.POST("/basket/add/:id", h.PostAddToDraft)
	r.POST("/basket/delete/:id", h.PostDeleteDraft)
	r.POST("/missions/:id/segments/:routeId/recalc", h.PostRecalcSegment)
	r.POST("/missions/:id/segments/:routeId/delete", h.PostDeleteSegment)

	// REST API для SPA (лаб. 3)
	apiGroup := r.Group("/api")
	{
		// Домен услуги
		apiGroup.GET("/services", h.APIGetServices)
		apiGroup.GET("/services/:id", h.APIGetService)
		apiGroup.POST("/services", h.APICreateService)

		// Домен м-м (услуги в заявке)
		apiGroup.POST("/requests/draft/items", h.APIAddToDraft)
		apiGroup.PUT("/requests/:id/items/:routeId", h.APIUpdateRequestItem)
		apiGroup.DELETE("/requests/:id/items/:routeId", h.APIDeleteRequestItem)

		// Домен заявки
		apiGroup.GET("/requests/cart-icon", h.APICartIcon)
		apiGroup.GET("/requests", h.APIGetRequests)
		apiGroup.GET("/requests/:id", h.APIGetRequest)
		apiGroup.PUT("/requests/:id", h.APIUpdateRequest)
		apiGroup.PUT("/requests/:id/form", h.APIFormRequest)
		apiGroup.PUT("/requests/:id/moderate", h.APIModerateRequest)
		apiGroup.DELETE("/requests/:id", h.APIDeleteRequest)

		// Домен пользователь
		apiGroup.POST("/users/register", h.APIRegisterUser)
		apiGroup.POST("/users/login", h.APILoginUser)
		apiGroup.POST("/users/logout", h.APILogoutUser)
	}

	r.Run()
	log.Println("Server down")
}
