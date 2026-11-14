package routes

import (
	_ "users-service/app/docs"
	"users-service/app/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterSwaggerRoutes(
	prefix string,
	router *gin.RouterGroup,
	tracker *middlewares.RequestTracker,
	middlewares middlewares.MiddlewaresInterface,
	logger zerolog.Logger,
) {
	logger.Info().Str("prefix", prefix).Msg("Registering Swagger routes")

	swaggerGroup := router.Group(prefix).
		Use(middlewares.Log(logger, "swagger")).
		Use(middlewares.LimitRequestsMiddleware(tracker))
	swaggerGroup.GET("/*any", func(c *gin.Context) {
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger.json"))(c)

		if c.Writer.Status() >= 400 {
			logger.Error().Int("status", c.Writer.Status()).Str("path", c.Request.URL.Path).Msg("Failed to serve Swagger resource")
		}
	})
}
