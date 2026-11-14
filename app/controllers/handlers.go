package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type StandardController interface {
	RespondFailure(ctx *gin.Context, statusCode int, errMsg string)
	RespondSuccess(ctx *gin.Context, statusCode int, data interface{})
	Log() *zerolog.Logger
}

func handleStandardRequest[T any, R any](
	ctx *gin.Context,
	c StandardController,
	serviceCall func(reqCtx context.Context, request *T) (R, error),
) {
	var req T
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.Log().Error().Err(err).Msg("Failed to bind JSON request")
		c.RespondFailure(ctx, http.StatusBadRequest, "invalid body request")
		return
	}

	type validatable interface {
		Valid() error
	}
	if v, ok := any(&req).(validatable); ok {
		if err := v.Valid(); err != nil {
			c.Log().Warn().Err(err).Msg("Invalid request data")
			c.RespondFailure(ctx, http.StatusBadRequest, "invalid request data: "+err.Error())
			return
		}
	}

	result, err := serviceCall(ctx.Request.Context(), &req)
	if err != nil {
		c.Log().Error().Err(err).Msg("Service call failed")
		c.RespondFailure(ctx, http.StatusUnprocessableEntity, err.Error())
		return
	}

	c.RespondSuccess(ctx, http.StatusOK, result)
}

func handleAuthenticatedRequest[T any, R any](
	ctx *gin.Context,
	c StandardController,
	serviceCall func(reqCtx context.Context, userID int, request *T) (R, error),
) {
	var req T
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.Log().Error().Err(err).Msg("Failed to bind JSON request")
		c.RespondFailure(ctx, http.StatusBadRequest, "Bad Request - Invalid input data")
		return
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		c.Log().Error().Err(err).Msg("Failed to get UserID from context")
		c.RespondFailure(ctx, http.StatusUnauthorized, "Unauthorized - Invalid or missing token")
		return
	}

	type validatable interface {
		Valid() error
	}
	if v, ok := any(&req).(validatable); ok {
		if err := v.Valid(); err != nil {
			c.Log().Warn().Err(err).Msg("Invalid request data")
			c.RespondFailure(ctx, http.StatusBadRequest, "Bad Request - Invalid input data")
			return
		}
	}

	result, err := serviceCall(ctx.Request.Context(), userID, &req)
	if err != nil {
		c.Log().Error().Err(err).Msg("Service call failed")
		c.RespondFailure(ctx, http.StatusUnprocessableEntity, "Unprocessable Entity - Update failed")
		return
	}

	c.RespondSuccess(ctx, http.StatusOK, result)
}

func getUserIDFromContext(ctx *gin.Context) (int, error) {
	userIDVal, exists := ctx.Get("UserID")
	if !exists {
		return 0, errors.New("UserID not found in context")
	}

	userID, ok := userIDVal.(int)
	if !ok {
		return 0, fmt.Errorf("UserID in context is not of type int, got %T", userIDVal)
	}

	return userID, nil
}
