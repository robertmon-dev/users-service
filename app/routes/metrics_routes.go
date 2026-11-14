package routes

import (
	"users-service/app/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

func RegisterMonitoringRoutes(
	prefix string,
	router *gin.RouterGroup,
	tracker *middlewares.RequestTracker,
	middlewares middlewares.MiddlewaresInterface,
	logger zerolog.Logger,
) {
	api := router.Group(prefix).
		Use(middlewares.Log(logger, "monitoring")).
		Use(middlewares.MetricsCORS()).
		Use(middlewares.LimitRequestsMiddleware(tracker))
	api.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
