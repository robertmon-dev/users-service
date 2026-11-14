//go:build unit

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"users-service/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleStandardRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name               string
		requestBody        interface{}
		serviceCall        func(reqCtx context.Context, req *mocks.TestRequest) (string, error)
		setupMocks         func(m *mocks.MockStandardController)
		expectedStatusCode int
	}{
		{
			name:        "Success",
			requestBody: mocks.TestRequest{Name: "test"},
			serviceCall: func(reqCtx context.Context, req *mocks.TestRequest) (string, error) {
				assert.Equal(t, "test", req.Name)
				return "success", nil
			},
			setupMocks: func(m *mocks.MockStandardController) {
				m.On("RespondSuccess", mock.Anything, http.StatusOK, "success").Once()
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:        "Failure - Invalid JSON",
			requestBody: "{invalid_json",
			setupMocks: func(m *mocks.MockStandardController) {
				m.On("Log").Return(&zerolog.Logger{}).Once()
				m.On("RespondFailure", mock.Anything, http.StatusBadRequest, "invalid body request").Once()
			},
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockController := new(mocks.MockStandardController)
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			var reqBodyBytes []byte
			if bodyStr, ok := tc.requestBody.(string); ok {
				reqBodyBytes = []byte(bodyStr)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.requestBody)
			}

			ctx.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBodyBytes))
			ctx.Request.Header.Set("Content-Type", "application/json")

			tc.setupMocks(mockController)

			handleStandardRequest(ctx, mockController, tc.serviceCall)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			mockController.AssertExpectations(t)
		})
	}
}

func TestHandleAuthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name               string
		setUserID          bool
		userID             int
		serviceCall        func(reqCtx context.Context, userID int, req *mocks.TestRequest) (string, error)
		setupMocks         func(m *mocks.MockStandardController)
		expectedStatusCode int
	}{
		{
			name:      "Success",
			setUserID: true,
			userID:    123,
			serviceCall: func(reqCtx context.Context, userID int, req *mocks.TestRequest) (string, error) {
				assert.Equal(t, 123, userID)
				assert.Equal(t, "auth_test", req.Name)
				return "authenticated success", nil
			},
			setupMocks: func(m *mocks.MockStandardController) {
				m.On("RespondSuccess", mock.Anything, http.StatusOK, "authenticated success").Once()
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:      "Bad Request",
			setUserID: true,
			userID:    123,
			serviceCall: func(reqCtx context.Context, userID int, req *mocks.TestRequest) (string, error) {
				assert.Equal(t, 123, userID)
				assert.Equal(t, "auth_test", req.Name)
				return "authenticated success", nil
			},
			setupMocks: func(m *mocks.MockStandardController) {
				m.On("RespondSuccess", mock.Anything, http.StatusOK, "authenticated success").Once()
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:      "Failure - Missing UserID in Context",
			setUserID: false,
			setupMocks: func(m *mocks.MockStandardController) {
				m.On("Log").Return(&zerolog.Logger{}).Once()
				m.On("RespondFailure", mock.Anything, http.StatusUnauthorized, "Unauthorized - Invalid or missing token").Once()
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockController := new(mocks.MockStandardController)
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			if tc.setUserID {
				ctx.Set("UserID", tc.userID)
			}

			reqBody := mocks.TestRequest{Name: "auth_test"}
			jsonBody, _ := json.Marshal(reqBody)
			ctx.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonBody))
			ctx.Request.Header.Set("Content-Type", "application/json")

			tc.setupMocks(mockController)

			handleAuthenticatedRequest(ctx, mockController, tc.serviceCall)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			mockController.AssertExpectations(t)
		})
	}
}
