package api

import (
	"log"

	"web_backend/internal/app/handler"
	"web_backend/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// StartServer инициализирует репозиторий, обработчики и запускает HTTP-сервер.
func StartServer() {
	log.Println("Starting server")

	repo, err := repository.NewRepository()
	if err != nil {
		logrus.Error("Ошибка инициализации репозитория")
	}

	h := handler.NewHandler(repo)

	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/", h.GetRoutes)
	r.GET("/routes/:id", h.GetRoute)
	r.GET("/missions/:id", h.GetMission)

	r.Run()
	log.Println("Server down")
}
