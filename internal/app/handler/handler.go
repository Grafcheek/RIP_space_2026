package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"web_backend/internal/app/repository"
)

// Handler содержит зависимости HTTP-обработчиков.
type Handler struct {
	Repository *repository.Repository
}

// NewHandler создаёт новый Handler с переданным репозиторием.
func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

// GetRoutes — главная: каталог interplanetary flights + корзина (interplanetary flight request).
func (h *Handler) GetRoutes(ctx *gin.Context) {
	searchQuery := ctx.Query("query")

	routes, err := h.Repository.SearchInterplanetaryFlights(searchQuery)
	if err != nil {
		logrus.Error(err)
	}

	interplanetaryFlightRequests, err := h.Repository.GetInterplanetaryFlightRequests()
	if err != nil {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"routes":                       routes,
		"query":                        searchQuery,
		"interplanetaryFlightRequests": interplanetaryFlightRequests,
	})
}

// GetRoute — детальная страница одного interplanetary flight + расчёт из строки interplanetary_flights_in_request.
func (h *Handler) GetRoute(ctx *gin.Context) {
	idStr := ctx.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	route, err := h.Repository.GetInterplanetaryFlightByID(id)
	if err != nil {
		logrus.Error(err)
	}

	flightInRequest, err := h.Repository.GetInterplanetaryFlightInRequestForFlight(id)
	hasFlightInRequestData := err == nil && flightInRequest != nil

	ctx.HTML(http.StatusOK, "strategy.html", gin.H{
		"route":                  route,
		"flightInRequest":        flightInRequest,
		"hasFlightInRequestData": hasFlightInRequestData,
	})
}

// GetInterplanetaryFlightRequest — страница заявки interplanetary_flights_requests и строк interplanetary_flights_in_request.
func (h *Handler) GetInterplanetaryFlightRequest(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	interplanetaryFlightRequest, err := h.Repository.GetInterplanetaryFlightRequest(id)
	if err != nil {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "interplanetary_flight_request.html", gin.H{
		"interplanetaryFlightRequest": interplanetaryFlightRequest,
	})
}
