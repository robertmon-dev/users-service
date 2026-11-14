package routes

import (
	"users-service/app/controllers"
	"users-service/app/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func RegisterDiagnosticsRoutes(prefix string,
	router *gin.RouterGroup,
	diagnosticsController controllers.DiagnosticsControllerInterface,
	tracker *middlewares.RequestTracker,
	middlewares middlewares.MiddlewaresInterface,
	logger zerolog.Logger,
) {
	api := router.Group(prefix).
		Use(middlewares.Log(logger, "diagnostics-controller")).
		Use(middlewares.LimitRequestsMiddleware(tracker))
	api.GET("/readiness", diagnosticsController.ReadinessProbe)
	api.GET("/health", diagnosticsController.HealthProbe)
}
