package services

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPasswordService(pepper string) *PasswordService {
	return NewPasswordService(zerolog.Nop(), []byte(pepper), 1)
}

func TestPasswordService_HashPassword(t *testing.T) {
	passwordService := newTestPasswordService("test-pepper")

	hash, err := passwordService.HashPassword("my-secret-password")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, "$argon2id$")
}

func TestPasswordService_CompareHashes(t *testing.T) {
	passwordService := newTestPasswordService("test-pepper")

	hash, err := passwordService.HashPassword("my-secret-password")
	require.NoError(t, err)

	testCases := []struct {
		name       string
		storedHash string
		password   string
		expectErr  bool
	}{
		{
			name:       "Success - Matching password",
			storedHash: hash,
			password:   "my-secret-password",
			expectErr:  false,
		},
		{
			name:       "Failure - Wrong password",
			storedHash: hash,
			password:   "wrong-password",
			expectErr:  true,
		},
		{
			name:       "Failure - Malformed stored hash",
			storedHash: "not-a-valid-hash",
			password:   "my-secret-password",
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := passwordService.CompareHashes(tc.storedHash, tc.password)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPasswordService_CompareHashes_DifferentPepper(t *testing.T) {
	hashingService := newTestPasswordService("pepper-one")
	comparingService := newTestPasswordService("pepper-two")

	hash, err := hashingService.HashPassword("my-secret-password")
	require.NoError(t, err)

	err = comparingService.CompareHashes(hash, "my-secret-password")
	require.Error(t, err)
}
