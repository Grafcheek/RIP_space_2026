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

	// REST API для SPA (лаб. 3) — пути в тематике interplanetary flight
	apiGroup := r.Group("/api")
	{
		// Домен межпланетного перелёта (услуга)
		apiGroup.GET("/interplanetaryflights", h.APIListInterplanetaryFlights)
		apiGroup.GET("/interplanetaryflights/:id", h.APIGetInterplanetaryFlight)
		apiGroup.POST("/interplanetaryflights", h.APICreateInterplanetaryFlight)

		// Домен м-м (перелёт в заявке)
		apiGroup.POST("/interplanetaryflightrequests/draft/items", h.APIAddInterplanetaryFlightToDraftRequest)
		apiGroup.PUT("/interplanetaryflightrequests/:id/items/:routeId", h.APIUpdateInterplanetaryFlightInRequest)
		apiGroup.DELETE("/interplanetaryflightrequests/:id/items/:routeId", h.APIDeleteInterplanetaryFlightInRequest)

		// Домен заявки на межпланетный перелёт (interplanetary flight request)
		apiGroup.GET("/interplanetaryflightrequests/cart-icon", h.APIGetInterplanetaryFlightRequestCartIcon)
		apiGroup.GET("/interplanetaryflightrequests", h.APIListInterplanetaryFlightRequests)
		apiGroup.GET("/interplanetaryflightrequests/:id", h.APIGetInterplanetaryFlightRequest)
		apiGroup.PUT("/interplanetaryflightrequests/:id", h.APIUpdateInterplanetaryFlightRequest)
		apiGroup.PUT("/interplanetaryflightrequests/:id/form", h.APIFormInterplanetaryFlightRequest)
		apiGroup.PUT("/interplanetaryflightrequests/:id/moderate", h.APIModerateInterplanetaryFlightRequest)
		apiGroup.DELETE("/interplanetaryflightrequests/:id", h.APIDeleteInterplanetaryFlightRequest)

		// Домен пользователя сервиса межпланетных перелётов
		apiGroup.POST("/interplanetaryflightusers/register", h.APIRegisterInterplanetaryFlightsUser)
		apiGroup.POST("/interplanetaryflightusers/login", h.APILoginInterplanetaryFlightsUser)
		apiGroup.POST("/interplanetaryflightusers/logout", h.APILogoutInterplanetaryFlightsUser)
	}

	r.Run()
	log.Println("Server down")
}
