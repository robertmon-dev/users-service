package services

import (
	"context"
	"errors"
	"time"

	"users-service/app/infrastructure"
	"users-service/app/models/ent"
	requests "users-service/app/models/requests"
	models "users-service/app/models/utils"
	"users-service/app/repositories"
	"users-service/app/utils"

	"github.com/rs/zerolog"
)

type AuthService struct {
	UsersRepository repositories.UsersRepositoryInterface
	EventNotifier   infrastructure.EventNotifierInterface
	Logger          zerolog.Logger
	BearerService   BearerServiceInterface
	PasswordService PasswordServiceInterface
}

type AuthServiceInterface interface {
	GenerateJWT(ctx context.Context, userID int, deviceID int, userRoles []models.UserRoleInfo) (string, error)
	ValidateJWT(ctx context.Context, tokenString string) (*models.Claims, error)
	HashPassword(password string) (string, error)
	CompareHashes(storedHash, password string) error

	Register(ctx context.Context, req *requests.Register) (*ent.User, error)
	Login(ctx context.Context, req *requests.Login) (*ent.User, string, error)

	ActivateAccount(ctx context.Context, req *requests.Code) (*ent.User, error)
	VerifyAccount(ctx context.Context, req *requests.Code) (*ent.User, string, error)
	VerifySecondFactor(ctx context.Context, req *requests.Code) (*ent.User, string, error)

	RequestPasswordReset(ctx context.Context, req *requests.Mail) error
	CancelPasswordReset(ctx context.Context, req *requests.Mail) error
	ConfirmPasswordReset(ctx context.Context, req *requests.ConfirmPasswordReset) error

	ResendActivationCode(ctx context.Context, req *requests.Mail) error
	ResendVerificationCode(ctx context.Context, req *requests.Mail) error
	ResendResetCode(ctx context.Context, req *requests.Mail) error
	ResendSecondFactorCode(ctx context.Context, req *requests.Mail) error
}

var _ AuthServiceInterface = (*AuthService)(nil)

func NewAuthService(
	usersRepository repositories.UsersRepositoryInterface,
	eventNotifier infrastructure.EventNotifierInterface,
	logger zerolog.Logger,
	bearerService BearerServiceInterface,
	passwordService PasswordServiceInterface,
) *AuthService {
	return &AuthService{
		UsersRepository: usersRepository,
		EventNotifier:   eventNotifier,
		Logger:          logger,
		BearerService:   bearerService,
		PasswordService: passwordService,
	}
}

func (s *AuthService) GenerateJWT(ctx context.Context, userID int, deviceID int, userRoles []models.UserRoleInfo) (string, error) {
	return s.BearerService.GenerateJWT(ctx, userID, deviceID, userRoles)
}

func (s *AuthService) ValidateJWT(ctx context.Context, tokenString string) (*models.Claims, error) {
	return s.BearerService.ValidateJWT(ctx, tokenString)
}

func (s *AuthService) Register(
	ctx context.Context,
	req *requests.Register,
) (*ent.User, error) {
	if user, err := s.UsersRepository.GetUserByMail(ctx, req.Mail); err == nil && user != nil {
		return nil, errors.New("this mail is already registered")
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, errors.New("failed to check for existing user")
	}

	hashedPassword, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash user's password")
	}

	newUser, activationCode, err := s.UsersRepository.CreateUser(ctx, req, hashedPassword)
	if err != nil {
		return nil, errors.New("failed to create new user")
	}

	if err := s.EventNotifier.CreateRegistrationEvent(newUser, activationCode); err != nil {
		s.Logger.Error().Err(err).Int("user_id", newUser.ID).Msg("Failed to produce registration event")
	}

	return newUser, nil
}

func (s *AuthService) ResendActivationCode(ctx context.Context, req *requests.Mail) error {
	user, err := s.UsersRepository.GetUserByMailWithCodes(ctx, req.Mail)
	if err != nil {
		return errors.New("failed to find user with the provided email")
	}

	if err := s.EventNotifier.CreateRegistrationEvent(user, user.Edges.ActivationCode); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce resend-activation event")
		return errors.New("failed to send activation code")
	}

	s.Logger.Info().Int("user_id", user.ID).Msg("Resent activation code event produced")
	return nil
}

func (s *AuthService) ActivateAccount(
	ctx context.Context,
	req *requests.Code,
) (*ent.User, error) {
	user, err := s.UsersRepository.ActivateAccount(ctx, req.Code)
	if err != nil {
		return nil, errors.New("provided code doesn't exist")
	}

	action := &ent.UserAction{Action: "account.activated"}
	if err := s.EventNotifier.CreateUserActionEvent(user.Edges.UserSettings, action); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce account activation event")
	}

	return user, nil
}

func (s *AuthService) HashPassword(password string) (string, error) {
	return s.PasswordService.HashPassword(password)
}

func (s *AuthService) CompareHashes(storedHash, password string) error {
	return s.PasswordService.CompareHashes(storedHash, password)
}

func (s *AuthService) Login(ctx context.Context, req *requests.Login) (*ent.User, string, error) {
	user, err := s.UsersRepository.GetUserByMail(ctx, req.Mail)
	if err != nil {
		return nil, "", err
	}

	if user.Blacklisted {
		return nil, "", errors.New("account is blacklisted")
	}

	if !user.Active {
		return nil, "", errors.New("account is not activated")
	}

	if err := s.CompareHashes(user.Password, req.Password); err != nil {
		return nil, "", errors.New("provided wrong password")
	}

	device, err := s.UsersRepository.FindOrCreateDevice(ctx, user.ID, &req.Device)
	if err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to find or create device")
		return nil, "", errors.New("failed to handle user device")
	}

	if user.Edges.UserSettings.TwoFactor {
		secondFactor, err := s.UsersRepository.CreateSecondFactorCode(ctx, user, device)
		if err != nil {
			return nil, "", errors.New("failed to create second factor code")
		}

		if err := s.EventNotifier.CreateSecondFactorEvent(user.Edges.UserSettings, secondFactor); err != nil {
			s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce second factor event")
			return nil, "", errors.New("failed to send second factor code")
		}

		return nil, "2fa_required", nil
	}

	userRoles := utils.MarshalUserRoles(user.Edges.UserRoles)
	token, err := s.GenerateJWT(ctx, user.ID, device.ID, userRoles)
	if err != nil {
		return nil, "", err
	}

	if err := s.EventNotifier.CreateLoginEvent(user.Edges.UserSettings, "password"); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce login event")
	}

	return user, token, nil
}

func (s *AuthService) ResendSecondFactorCode(ctx context.Context, req *requests.Mail) error {
	user, err := s.UsersRepository.GetUserByMail(ctx, req.Mail)
	if err != nil {
		return errors.New("failed to find user with the provided email")
	}

	if !user.Edges.UserSettings.TwoFactor {
		return errors.New("second factor is not enabled for this account")
	}

	if user.Edges.SecondFactorCode == nil {
		return errors.New("user has no login attempts created")
	}

	if err := s.EventNotifier.CreateSecondFactorEvent(user.Edges.UserSettings, user.Edges.SecondFactorCode); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce resend-second-factor event")
		return errors.New("failed to send second factor code")
	}

	s.Logger.Info().Int("user_id", user.ID).Msg("Resent second factor event produced")
	return nil
}

func (s *AuthService) VerifySecondFactor(ctx context.Context, req *requests.Code) (*ent.User, string, error) {
	foundUser, err := s.UsersRepository.GetUserBySecondFactor(ctx, req.Code)
	if err != nil {
		return nil, "", errors.New("failed to fetch user by second factor code")
	}

	if time.Now().UTC().After(foundUser.Edges.SecondFactorCode.ExpiresAt) {
		go func() {
			if _, err := s.UsersRepository.RemoveSecondFactorCode(context.Background(), foundUser); err != nil {
				s.Logger.Error().Err(err).Int("user_id", foundUser.ID).Msg("Failed to remove expired second factor code")
			}
		}()
		return nil, "", errors.New("second factor token has expired")
	}

	device := foundUser.Edges.SecondFactorCode.Edges.TargetUserDevice
	if device == nil {
		s.Logger.Error().Int("user_id", foundUser.ID).Msg("Device context not found for second factor verification")
		return nil, "", errors.New("could not determine the device for this session")
	}

	userRoles := utils.MarshalUserRoles(foundUser.Edges.UserRoles)
	token, err := s.GenerateJWT(ctx, foundUser.ID, device.ID, userRoles)
	if err != nil {
		return nil, "", errors.New("failed to create Bearer for user")
	}

	user, err := s.UsersRepository.RemoveSecondFactorCode(ctx, foundUser)
	if err != nil {
		s.Logger.Error().Err(err).Int("user_id", foundUser.ID).Msg("Failed to remove second factor code after use")
		user = foundUser
	}

	if err := s.EventNotifier.CreateLoginEvent(foundUser.Edges.UserSettings, "second-factor"); err != nil {
		s.Logger.Error().Err(err).Int("user_id", foundUser.ID).Msg("Failed to produce login event after 2FA")
	}

	return user, token, nil
}

func (s *AuthService) ResendVerificationCode(ctx context.Context, req *requests.Mail) error {
	user, err := s.UsersRepository.GetUserByMailWithCodes(ctx, req.Mail)
	if err != nil {
		return errors.New("failed to find user with the provided email")
	}

	if err := s.EventNotifier.CreateVerificationEvent(user.Edges.UserSettings, user.Phone, user.Edges.VerificationCode); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce resend-verification event")
		return errors.New("failed to send verification code")
	}

	s.Logger.Info().Int("user_id", user.ID).Msg("Resent verification code event produced")
	return nil
}

func (s *AuthService) VerifyAccount(ctx context.Context, req *requests.Code) (*ent.User, string, error) {
	user, err := s.UsersRepository.VerifyAccount(ctx, req.Code)
	if err != nil {
		return nil, "", errors.New("failed to fetch user by verification code")
	}

	device, err := s.UsersRepository.FindOrCreateDevice(ctx, user.ID, &req.Device)
	if err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to find or create device during account verification")
		return nil, "", errors.New("failed to handle user device")
	}

	userRoles := utils.MarshalUserRoles(user.Edges.UserRoles)
	token, err := s.GenerateJWT(ctx, user.ID, device.ID, userRoles)
	if err != nil {
		return nil, "", errors.New("failed to create Bearer for user")
	}

	return user, token, nil
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, req *requests.Mail) error {
	user, err := s.UsersRepository.GetUserByMail(ctx, req.Mail)
	if err != nil {
		return errors.New("failed to fetch user")
	}

	resetCode, err := s.UsersRepository.CreateResetCode(ctx, user)
	if err != nil {
		return errors.New("failed to create reset token")
	}

	if err := s.EventNotifier.CreateResetPasswordEvent(user.Edges.UserSettings, resetCode); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce reset password event")
		return errors.New("failed to send reset password code")
	}

	return nil
}

func (s *AuthService) CancelPasswordReset(ctx context.Context, req *requests.Mail) error {
	user, err := s.UsersRepository.GetUserByMail(ctx, req.Mail)
	if err != nil {
		return errors.New("failed to fetch user")
	}

	if _, err = s.UsersRepository.RemoveResetCode(ctx, user); err != nil {
		return errors.New("failed to cancel password reset process")
	}

	action := &ent.UserAction{Action: "password.reset.cancelled"}
	if err := s.EventNotifier.CreateUserActionEvent(user.Edges.UserSettings, action); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce password reset cancellation event")
	}

	return nil
}

func (s *AuthService) ConfirmPasswordReset(ctx context.Context, req *requests.ConfirmPasswordReset) error {
	user, err := s.UsersRepository.GetUserByResetCode(ctx, req.Code)
	if err != nil {
		return errors.New("failed to fetch user by reset code")
	}

	hashedPassword, err := s.HashPassword(req.Password)
	if err != nil {
		return errors.New("failed to hash user's password")
	}

	if _, err = s.UsersRepository.UpdateUsersPassword(ctx, user, hashedPassword); err != nil {
		return errors.New("failed to update users password")
	}

	if _, err = s.UsersRepository.RemoveResetCode(ctx, user); err != nil {
		return errors.New("failed to remove reset code")
	}

	action := &ent.UserAction{Action: "password.reset.confirmed"}
	if err := s.EventNotifier.CreateUserActionEvent(user.Edges.UserSettings, action); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce password reset confirmation event")
	}

	return nil
}

func (s *AuthService) ResendResetCode(ctx context.Context, req *requests.Mail) error {
	user, err := s.UsersRepository.GetUserByMailWithCodes(ctx, req.Mail)
	if err != nil {
		return errors.New("failed to find user with the provided email")
	}

	resetCode, err := s.UsersRepository.CreateResetCode(ctx, user)
	if err != nil {
		return errors.New("failed to create a new reset code")
	}

	if err := s.EventNotifier.CreateResetPasswordEvent(user.Edges.UserSettings, resetCode); err != nil {
		s.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to produce resend-reset-code event")
		return errors.New("failed to send reset code")
	}

	s.Logger.Info().Int("user_id", user.ID).Msg("Resent reset code event produced")
	return nil
}
