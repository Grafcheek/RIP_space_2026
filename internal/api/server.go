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

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/", h.GetRoutes)
	r.GET("/routes/:id", h.GetRoute)
	r.GET("/missions/:id", h.GetMission)
	r.POST("/basket/add/:id", h.PostAddToDraft)
	r.POST("/basket/delete/:id", h.PostDeleteDraft)
	r.POST("/missions/:id/segments/:routeId/recalc", h.PostRecalcSegment)
	r.POST("/missions/:id/segments/:routeId/delete", h.PostDeleteSegment)

	r.Run()
	log.Println("Server down")
}
