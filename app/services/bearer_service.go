package services

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"time"

	models "users-service/app/models/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

type BearerService struct {
	Logger        zerolog.Logger
	PublicKey     crypto.PublicKey
	PrivateKey    crypto.PrivateKey
	TokenDuration time.Duration
}

type BearerServiceInterface interface {
	GenerateJWT(ctx context.Context, userID int, deviceID int, userRoles []models.UserRoleInfo) (string, error)
	ValidateJWT(ctx context.Context, tokenString string) (*models.Claims, error)
}

var _ BearerServiceInterface = (*BearerService)(nil)

func NewBearerService(
	logger zerolog.Logger,
	publicKey crypto.PublicKey,
	privateKey crypto.PrivateKey,
	tokenDuration time.Duration,
) *BearerService {
	return &BearerService{
		Logger:        logger,
		PublicKey:     publicKey,
		PrivateKey:    privateKey,
		TokenDuration: tokenDuration,
	}
}

func (s *BearerService) GenerateJWT(ctx context.Context, userID int, deviceID int, userRoles []models.UserRoleInfo) (string, error) {
	s.Logger.Info().Int("user_id", userID).Int("device_id", deviceID).Msg("Generating JWT for device")

	claims := models.Claims{
		UserID:    userID,
		DeviceID:  deviceID,
		UserRoles: userRoles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(s.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}

	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, claims)
	resultChan := make(chan models.JWTResult, 1)

	go func() {
		defer close(resultChan)
		signedToken, err := token.SignedString(s.PrivateKey)
		if err != nil {
			s.Logger.Error().Int("user_id", userID).Err(err).Msg("Failed to sign JWT")
			resultChan <- models.JWTResult{Token: "", Err: err}
			return
		}
		s.Logger.Info().Int("user_id", userID).Msg("JWT successfully generated")
		resultChan <- models.JWTResult{Token: signedToken, Err: nil}
	}()

	select {
	case result := <-resultChan:
		if result.Err != nil {
			return "", result.Err
		}
		return result.Token, nil
	case <-ctx.Done():
		s.Logger.Warn().Int("user_id", userID).Err(ctx.Err()).Msg("JWT generation timed out")
		return "", errors.New("JWT generation timed out")
	}
}

func (s *BearerService) ValidateJWT(ctx context.Context, tokenString string) (*models.Claims, error) {
	s.Logger.Info().Msg("Validating JWT")

	resultChan := make(chan models.JWTValidationResult, 1)

	go func() {
		defer close(resultChan)
		token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
				s.Logger.Warn().Str("alg", fmt.Sprintf("%v", token.Header["alg"])).Msg("Unexpected signing method")
				return nil, errors.New("unexpected signing method")
			}
			return s.PublicKey, nil
		})
		if err != nil {
			s.Logger.Error().Err(err).Msg("JWT parsing failed")
			resultChan <- models.JWTValidationResult{Claims: nil, Err: err}
			return
		}

		claims, ok := token.Claims.(*models.Claims)
		if !ok || !token.Valid {
			s.Logger.Warn().Msg("Invalid token or claims mismatch")
			resultChan <- models.JWTValidationResult{Claims: nil, Err: errors.New("invalid token or claims")}
			return
		}

		if claims.DeviceID == 0 {
			s.Logger.Warn().Int("user_id", claims.UserID).Msg("Token is missing device ID")
			resultChan <- models.JWTValidationResult{Claims: nil, Err: errors.New("token is not device-specific")}
			return
		}

		if claims.ExpiresAt != nil && claims.ExpiresAt.Unix() <= time.Now().UTC().Unix() {
			s.Logger.Warn().
				Int("user_id", claims.UserID).
				Int("device_id", claims.DeviceID).
				Time("exp", claims.ExpiresAt.Time).
				Msg("Token expired")
			resultChan <- models.JWTValidationResult{Claims: nil, Err: errors.New("token has expired")}
			return
		}

		s.Logger.Info().
			Int("user_id", claims.UserID).
			Int("device_id", claims.DeviceID).
			Interface("userRoles", claims.UserRoles).
			Msg("Token validated successfully")
		resultChan <- models.JWTValidationResult{Claims: claims, Err: nil}
	}()

	select {
	case result := <-resultChan:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Claims, nil
	case <-ctx.Done():
		s.Logger.Warn().Err(ctx.Err()).Msg("JWT validation timed out")
		return nil, errors.New("JWT validation timed out")
	}
}
