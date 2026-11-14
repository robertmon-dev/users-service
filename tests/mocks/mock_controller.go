package mocks

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

type MockStandardController struct {
	mock.Mock
}

func (m *MockStandardController) RespondFailure(ctx *gin.Context, statusCode int, errMsg string) {
	m.Called(ctx, statusCode, errMsg)
	ctx.JSON(statusCode, gin.H{"error": errMsg})
}

func (m *MockStandardController) RespondSuccess(ctx *gin.Context, statusCode int, data interface{}) {
	m.Called(ctx, statusCode, data)
	ctx.JSON(statusCode, gin.H{"data": data})
}

func (m *MockStandardController) Log() *zerolog.Logger {
	args := m.Called()
	return args.Get(0).(*zerolog.Logger)
}

type TestRequest struct {
	Name string `json:"name"`
}

type TestRequestWithValidation struct {
	Name string `json:"name"`
}

func (r *TestRequestWithValidation) Valid() error {
	if r.Name == "invalid" {
		return errors.New("name is invalid")
	}
	return nil
}
