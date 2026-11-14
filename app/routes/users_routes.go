package routes

import (
	"users-service/app/controllers"
	"users-service/app/middlewares"
	"users-service/app/services"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func RegisterUserRoutes(
	prefix string,
	router *gin.RouterGroup,
	usersController controllers.UsersControllerInterface,
	authService services.AuthServiceInterface,
	tracker *middlewares.RequestTracker,
	middlewares middlewares.MiddlewaresInterface,
	logger zerolog.Logger,
) {
	api := router.Group(prefix).
		Use(middlewares.Log(logger, "users-controller")).
		Use(middlewares.LimitRequestsMiddleware(tracker)).
		Use(middlewares.UserAuthenticationMiddleware(authService))

	api.PATCH("/update", usersController.UpdateUser)
	api.PATCH("/details/update", usersController.UpdateDetails)
	api.PATCH("/settings/update", usersController.UpdateSettings)
	api.DELETE("/remove", usersController.RemoveAccount)
}
