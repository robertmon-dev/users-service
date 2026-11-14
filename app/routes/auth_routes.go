package routes

import (
	"users-service/app/controllers"
	"users-service/app/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func RegisterAuthRoutes(
	prefix string,
	router *gin.RouterGroup,
	authController controllers.AuthControllerInterface,
	tracker *middlewares.RequestTracker,
	middlewares middlewares.MiddlewaresInterface,
	logger zerolog.Logger,
) {
	api := router.Group(prefix).
		Use(middlewares.Log(logger, "auth-controller")).
		Use(middlewares.LimitRequestsMiddleware(tracker))
	api.POST("/register", authController.Register)
	api.POST("/register/externally", authController.RegisterExternally)
	api.PATCH("/activation/account", authController.ActivateAccount)
	api.POST("/activation/resend/code", authController.ResendActivationCode)
	api.PATCH("/verification/account", authController.VerifyAccount)
	api.POST("/verification/resend/code", authController.ResendVerificationCode)
	api.POST("/login", authController.Login)
	api.POST("/login/externally", authController.LoginExternally)
	api.DELETE("/logout", authController.Logout)
	api.POST("/second-factor/resend/code", authController.ResendSecondFactorCode)
	api.POST("/second-factor/verification/code", authController.VerifySecondFactorCode)
	api.POST("/password/reset/request", authController.RequestPasswordReset)
	api.DELETE("/password/reset/request/cancel", authController.CancelPasswordReset)
	api.PATCH("/password/reset/request/confirm", authController.ConfirmPasswordReset)
	api.POST("/password/reset/request/resend", authController.ResendResetCode)
}
