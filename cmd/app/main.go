// @title           Interplanetary Flight API
// @version         1.0
// @description     API заявок на расчёт межпланетного перелёта: характеристическая скорость (Δv), масса топлива, энергия.
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
package main

import (
	"log"

	_ "web_backend/docs"

	"web_backend/internal/api"
)

func main() {
	log.Println("Application start!")
	api.StartServer()
	log.Println("Application terminated!")
}
