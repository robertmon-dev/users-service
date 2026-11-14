package services

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	models "users-service/app/models/utils"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBearerService(t *testing.T, tokenDuration time.Duration) (*BearerService, ed25519.PublicKey, ed25519.PrivateKey) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	return NewBearerService(zerolog.Nop(), pubKey, privKey, tokenDuration), pubKey, privKey
}

func TestBearerService_GenerateJWT(t *testing.T) {
	bearerService, _, _ := newTestBearerService(t, 15*time.Minute)

	userRoles := []models.UserRoleInfo{{Name: "admin"}}
	token, err := bearerService.GenerateJWT(context.Background(), 1, 2, userRoles)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestBearerService_ValidateJWT(t *testing.T) {
	testCases := []struct {
		name          string
		tokenDuration time.Duration
		deviceID      int
		useOtherKey   bool
		expectErr     bool
		expectedErr   string
	}{
		{
			name:          "Success",
			tokenDuration: 15 * time.Minute,
			deviceID:      1,
			expectErr:     false,
		},
		{
			name:          "Failure - Expired token",
			tokenDuration: -15 * time.Minute,
			deviceID:      1,
			expectErr:     true,
			expectedErr:   "expired",
		},
		{
			name:          "Failure - Missing device ID",
			tokenDuration: 15 * time.Minute,
			deviceID:      0,
			expectErr:     true,
			expectedErr:   "token is not device-specific",
		},
		{
			name:          "Failure - Signed with different key",
			tokenDuration: 15 * time.Minute,
			deviceID:      1,
			useOtherKey:   true,
			expectErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bearerService, pubKey, privKey := newTestBearerService(t, tc.tokenDuration)

			signingKey := privKey
			if tc.useOtherKey {
				_, otherPrivKey, err := ed25519.GenerateKey(nil)
				require.NoError(t, err)
				signingKey = otherPrivKey
			}

			signingService := NewBearerService(zerolog.Nop(), pubKey, signingKey, tc.tokenDuration)
			token, err := signingService.GenerateJWT(context.Background(), 1, tc.deviceID, nil)
			require.NoError(t, err)

			claims, err := bearerService.ValidateJWT(context.Background(), token)

			if tc.expectErr {
				require.Error(t, err)
				if tc.expectedErr != "" {
					assert.Contains(t, err.Error(), tc.expectedErr)
				}
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				assert.Equal(t, 1, claims.UserID)
				assert.Equal(t, tc.deviceID, claims.DeviceID)
			}
		})
	}
}
