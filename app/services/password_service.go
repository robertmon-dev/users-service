package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/argon2"
)

type PasswordService struct {
	Logger        zerolog.Logger
	HashingPepper []byte
	HashPool      chan func()
}

type PasswordServiceInterface interface {
	HashPassword(password string) (string, error)
	CompareHashes(storedHash, password string) error
}

var _ PasswordServiceInterface = (*PasswordService)(nil)

func NewPasswordService(
	logger zerolog.Logger,
	hashingPepper []byte,
	poolSize int,
) *PasswordService {
	passwordService := &PasswordService{
		Logger:        logger,
		HashingPepper: hashingPepper,
		HashPool:      make(chan func(), poolSize),
	}

	for i := 0; i < poolSize; i++ {
		go func() {
			for task := range passwordService.HashPool {
				task()
			}
		}()
	}

	return passwordService
}

func (s *PasswordService) HashPassword(password string) (string, error) {
	resultChan := make(chan struct {
		Hash string
		Err  error
	}, 1)

	s.HashPool <- func() {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			s.Logger.Error().Err(err).Msg("Failed to generate salt for password hashing")
			resultChan <- struct {
				Hash string
				Err  error
			}{"", err}
			return
		}

		h := hmac.New(sha256.New, s.HashingPepper)
		h.Write([]byte(password))
		hmacValue := h.Sum(nil)

		hashed := argon2.IDKey(hmacValue, salt, 3, 128*1024, 4, 32)
		hashedStr := fmt.Sprintf("$argon2id$v=19$m=131072,t=3,p=4$%s$%s",
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(hashed))

		resultChan <- struct {
			Hash string
			Err  error
		}{hashedStr, nil}
	}

	result := <-resultChan
	return result.Hash, result.Err
}

func (s *PasswordService) CompareHashes(storedHash, password string) error {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 6 {
		return errors.New("invalid stored hash format")
	}

	var memory, hashTime uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &hashTime, &threads)
	if err != nil {
		return fmt.Errorf("failed to parse hash parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("failed to decode salt from hash: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("failed to decode hash part: %w", err)
	}

	resultChan := make(chan error, 1)

	s.HashPool <- func() {
		h := hmac.New(sha256.New, s.HashingPepper)
		h.Write([]byte(password))
		hmacValue := h.Sum(nil)

		comparisonHash := argon2.IDKey(hmacValue, salt, hashTime, memory, threads, uint32(len(hash)))

		if subtle.ConstantTimeCompare(hash, comparisonHash) == 1 {
			resultChan <- nil
		} else {
			resultChan <- errors.New("invalid credentials")
		}
	}

	return <-resultChan
}
