package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterAPIRoutes настраивает REST /api с разделением гость / пользователь / модератор и Swagger UI.
func (h *Handler) RegisterAPIRoutes(router *gin.Engine) {
	router.Use(CORSMiddleware())

	api := router.Group("/api")

	// Гость: только чтение услуг (interplanetary flight) + регистрация и вход.
	pub := api.Group("/")
	pub.POST("/interplanetaryflightusers/register", h.APIRegisterInterplanetaryFlightsUser)
	pub.POST("/interplanetaryflightusers/login", h.APILoginInterplanetaryFlightsUser)
	pub.GET("/interplanetaryflights", h.APIListInterplanetaryFlights)
	pub.GET("/interplanetaryflights/:id", h.APIGetInterplanetaryFlight)

	opt := api.Group("/")
	opt.Use(h.WithOptionalAuthCheck())
	opt.GET("/interplanetaryflightrequests/cart-icon", h.APIGetInterplanetaryFlightRequestCartIcon)

	auth := api.Group("/")
	auth.Use(h.JWTAuthMiddleware(false))
	auth.POST("/interplanetaryflights", h.APICreateInterplanetaryFlight)
	auth.POST("/interplanetaryflightrequests/draft/items", h.APIAddInterplanetaryFlightToDraftRequest)
	auth.PUT("/interplanetaryflightrequests/:id/items/:routeId", h.APIUpdateInterplanetaryFlightInRequest)
	auth.DELETE("/interplanetaryflightrequests/:id/items/:routeId", h.APIDeleteInterplanetaryFlightInRequest)
	auth.GET("/interplanetaryflightrequests", h.APIListInterplanetaryFlightRequests)
	auth.GET("/interplanetaryflightrequests/:id", h.APIGetInterplanetaryFlightRequest)
	auth.PUT("/interplanetaryflightrequests/:id", h.APIUpdateInterplanetaryFlightRequest)
	auth.PUT("/interplanetaryflightrequests/:id/form", h.APIFormInterplanetaryFlightRequest)
	auth.DELETE("/interplanetaryflightrequests/:id", h.APIDeleteInterplanetaryFlightRequest)
	auth.POST("/interplanetaryflightusers/logout", h.APILogoutInterplanetaryFlightsUser)

	mod := api.Group("/")
	mod.Use(h.JWTAuthMiddleware(true))
	mod.PUT("/interplanetaryflightrequests/:id/moderate", h.APIModerateInterplanetaryFlightRequest)

	swaggerURL := ginSwagger.URL("/swagger/doc.json")
	router.Any("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
}
