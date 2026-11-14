package repositories

import (
	"context"
	"errors"
	"strconv"

	"users-service/app/models/ent"
	"users-service/app/models/ent/activationcode"
	"users-service/app/models/ent/resetcode"
	"users-service/app/models/ent/secondfactorcode"
	"users-service/app/models/ent/user"
	"users-service/app/models/ent/verificationcode"
	"users-service/app/models/requests"
	"users-service/app/utils"

	"github.com/google/uuid"
)

func (r *UsersRepository) ActivateAccount(ctx context.Context, code string) (*ent.User, error) {
	r.Logger.Info().Str("activation_code", code).Msg("Attempting to activate account")
	var updatedUser *ent.User

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err := GetUserByActivationCode(ctx, tx, code)
		if err != nil {
			r.Logger.Error().Err(err).Msg("Failed to get user by activation code within transaction")
			return err
		}
		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User found for activation")

		if _, err = foundUser.
			Update().
			SetActive(true).
			Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", foundUser.ID).Msg("Failed to set user active")
			return err
		}
		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User account set to active")

		if _, err := tx.ActivationCode.
			Delete().
			Where(activationcode.CodeEQ(code)).
			Exec(ctx); err != nil {
			r.Logger.Error().Msg("Failed to delete activation code")
			return err
		}
		r.Logger.Debug().Str("activation_code", code).Msg("Activation code deleted")

		updatedUser, err = getUserWithSettings(ctx, tx, user.IDEQ(foundUser.ID))
		if err != nil {
			return errors.New("failed to find user's settings")
		}

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("activation_code", code).Msg("Transaction failed for ActivateAccount")
		return nil, err
	}

	InvalidateCache(ctx, updatedUser, r)

	r.Logger.Info().Int("user_id", updatedUser.ID).Msg("Account successfully activated")
	return updatedUser, nil
}

func (r *UsersRepository) VerifyAccount(ctx context.Context, code string) (*ent.User, error) {
	r.Logger.Info().Str("verification_code", code).Msg("Attempting to verify account")
	var foundUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByVerificationCode(ctx, tx, code)
		if err != nil {
			r.Logger.Error().Err(err).Str("verification_code", code).Msg("Failed to get user by verification code within transaction")
			return err
		}
		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User found for verification")

		foundUser, err = tx.User.
			UpdateOneID(foundUser.ID).
			SetVerified(true).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", foundUser.ID).Msg("Failed to set user verified")
			return err
		}
		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User account set to verified")

		if _, err := tx.VerificationCode.
			Delete().
			Where(verificationcode.CodeEQ(code)).
			Exec(ctx); err != nil {
			r.Logger.Error().Err(err).Str("verification_code", code).Msg("Failed to delete verification code")
			return err
		}

		foundUser, _ = getUserWithRoles(ctx, tx, user.IDEQ(foundUser.ID))

		r.Logger.Debug().Str("verification_code", code).Msg("Verification code deleted")

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("verification_code", code).Msg("Transaction failed for VerifyAccount")
		return nil, err
	}

	InvalidateCache(ctx, foundUser, r)

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Account successfully verified")
	return foundUser, nil
}

func (r *UsersRepository) CreateUser(ctx context.Context, req *requests.Register, hashedPassword string) (*ent.User, *ent.ActivationCode, error) {
	r.Logger.Info().Str("mail", req.Mail).Msg("Attempting to create new user")
	var newUser *ent.User
	var activationCode *ent.ActivationCode
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		newUser, err = tx.User.
			Create().
			SetMail(req.Mail).
			SetPassword(hashedPassword).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Str("mail", req.Mail).Msg("Failed to create user")
			return err
		}
		r.Logger.Debug().Int("user_id", newUser.ID).Msg("User created successfully")

		if _, err := tx.UserSettings.
			Create().
			SetUUID(uuid.New().String()).
			SetOwner(newUser).
			Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", newUser.ID).Msg("Failed to create user settings")
			return errors.New("failed to create user settings")
		}
		r.Logger.Debug().Int("user_id", newUser.ID).Msg("User settings created")

		if _, err := tx.UserDetails.
			Create().
			SetName(req.Name).
			SetOwner(newUser).
			Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", newUser.ID).Msg("Failed to create user details")
			return errors.New("failed to create user details")
		}
		r.Logger.Debug().Int("user_id", newUser.ID).Msg("User details created")

		if _, err := tx.UserDevice.
			Create().
			SetOwner(newUser).
			SetToken(req.DeviceToken).
			SetIPAddress(req.IPAddress).
			SetUserAgent(req.UserAgent).
			SetOsName(req.OSName).
			SetOsVersion(req.OSVersion).
			SetBrowserName(req.BrowserName).
			SetBrowserVersion(req.BrowserVersion).
			Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", newUser.ID).Msg("Failed to create user device")
			return errors.New("failed to create user device")
		}
		r.Logger.Debug().Int("user_id", newUser.ID).Str("device_token", req.DeviceToken).Msg("User device created")

		var codeStr string
		for {
			code := utils.GenerateRandomCode()
			codeStr = strconv.Itoa(code)

			if exists, _ := tx.ActivationCode.
				Query().
				Where(activationcode.CodeEQ(codeStr)).
				Exist(ctx); !exists {
				r.Logger.Debug().Str("activation_code", codeStr).Msg("Generated unique activation code")
				break
			}
			r.Logger.Debug().Str("activation_code", codeStr).Msg("Generated activation code already exists, retrying")
		}

		activationCode, err = tx.ActivationCode.
			Create().
			SetCode(codeStr).
			SetOwner(newUser).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", newUser.ID).Msg("Failed to create activation code")
			return err
		}
		r.Logger.Debug().Int("activation_code_id", activationCode.ID).Msg("Activation code created")

		newUser, err = getUserWithCodes(ctx, tx, user.IDEQ(newUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("mail", req.Mail).Msg("Transaction failed for CreateUser")
		return nil, nil, err
	}

	r.Logger.Info().Int("user_id", newUser.ID).Msg("User and associated entities created successfully")
	return newUser, activationCode, nil
}

func (r *UsersRepository) CreateSecondFactorCode(
	ctx context.Context,
	user *ent.User,
	device *ent.UserDevice,
) (*ent.SecondFactorCode, error) {
	r.Logger.Info().Int("user_id", user.ID).Int("device_id", device.ID).Msg("Attempting to create second factor code")
	var secondFactor *ent.SecondFactorCode
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		var codeStr string
		for {
			code := utils.GenerateRandomCode()
			codeStr = strconv.Itoa(code)

			exists, err := tx.SecondFactorCode.
				Query().
				Where(secondfactorcode.CodeEQ(codeStr)).
				Exist(ctx)
			if err != nil {
				r.Logger.Error().Err(err).Msg("Failed to check existence of second factor code")
				return err
			}
			if !exists {
				r.Logger.Debug().Str("second_factor_id", codeStr).Msg("Generated unique second factor code")
				break
			}
			r.Logger.Debug().Str("second_factor_id", codeStr).Msg("Generated second factor code already exists, retrying")
		}

		secondFactor, err = tx.SecondFactorCode.
			Create().
			SetCode(codeStr).
			SetOwner(user).
			SetTargetUserDeviceID(device.ID).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to create second factor code")
			return err
		}
		r.Logger.Debug().Int("second_factor_id", secondFactor.ID).Msg("Second factor code created")

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for CreateSecondFactorCode")
		return nil, err
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Info().Int("user_id", user.ID).Msg("Second factor code created successfully")
	return secondFactor, nil
}

func (r *UsersRepository) CreateResetCode(ctx context.Context, user *ent.User) (*ent.ResetCode, error) {
	r.Logger.Info().Int("user_id", user.ID).Msg("Attempting to create reset code")
	var resetCode *ent.ResetCode
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		var codeStr string

		for {
			code := utils.GenerateRandomCode()
			codeStr = strconv.Itoa(code)

			if exists, _ := tx.ResetCode.
				Query().
				Where(resetcode.CodeEQ(codeStr)).
				Exist(ctx); !exists {
				r.Logger.Debug().Str("reset_code", codeStr).Msg("Generated unique reset code")
				break
			}
			r.Logger.Debug().Str("reset_code", codeStr).Msg("Generated reset code already exists, retrying")
		}

		resetCode, err = tx.ResetCode.
			Create().
			SetCode(codeStr).
			SetOwner(user).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to create reset code")
			return err
		}
		r.Logger.Debug().Int("reset_code_id", resetCode.ID).Msg("Reset code created")

		r.Logger.Debug().Int("user_id", user.ID).Msg("User re-fetched after reset code creation (for consistency check)")

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for CreateResetCode")
		return nil, err
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Info().Int("user_id", user.ID).Msg("Reset code created successfully")
	return resetCode, nil
}

func (r *UsersRepository) RemoveResetCode(ctx context.Context, user *ent.User) (*ent.User, error) {
	r.Logger.Info().Int("user_id", user.ID).Msg("Attempting to remove reset code")
	var updatedUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		if user.Edges.ResetCode == nil {
			r.Logger.Warn().Int("user_id", user.ID).Msg("User has no reset code to remove")
			return errors.New("user has no reset code")
		}
		r.Logger.Debug().Int("reset_code_id", user.Edges.ResetCode.ID).Msg("Reset code found for deletion")

		if err := tx.ResetCode.
			DeleteOneID(user.Edges.ResetCode.ID).
			Exec(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", user.ID).Int("reset_code_id", user.Edges.ResetCode.ID).Msg("Failed to delete reset code")
			return err
		}
		r.Logger.Debug().Int("reset_code_id", user.Edges.ResetCode.ID).Msg("Reset code deleted")

		updatedUser, err = GetUserByID(ctx, tx, user.ID)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to re-fetch user after reset code removal")
			return err
		}
		r.Logger.Debug().Int("user_id", updatedUser.ID).Msg("User re-fetched after reset code removal")

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for RemoveResetCode")
		return nil, err
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Info().Int("user_id", updatedUser.ID).Msg("Reset code removed successfully")
	return updatedUser, nil
}

func (r *UsersRepository) RemoveSecondFactorCode(ctx context.Context, targetUser *ent.User) (*ent.User, error) {
	r.Logger.Info().Int("user_id", targetUser.ID).Msg("Attempting to remove second factor code")
	var updatedUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		if targetUser.Edges.SecondFactorCode == nil {
			r.Logger.Warn().Int("user_id", targetUser.ID).Msg("User has no second factor code to remove")
			return errors.New("user has no reset code")
		}

		r.Logger.Debug().Int("second_factor_code_id", targetUser.Edges.SecondFactorCode.ID).Msg("Second factor code found for deletion")

		updatedUser, err = tx.User.
			UpdateOne(targetUser).
			ClearSecondFactorCode().
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Int("second_factor_code_id", targetUser.Edges.SecondFactorCode.ID).Msg("Failed to remove second factor code")
			return errors.New("failed to remove second factor code")
		}
		r.Logger.Debug().Int("second_factor_code_id", targetUser.Edges.SecondFactorCode.ID).Msg("Second factor code deleted")

		updatedUser, err = getUserFunctional(ctx, tx, user.IDEQ(targetUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Transaction failed for RemoveSecondFactorCode")
		return nil, err
	}

	InvalidateCache(ctx, targetUser, r)

	return updatedUser, nil
}

func (r *UsersRepository) UpdateUsersPassword(ctx context.Context, user *ent.User, hashedPassword string) (*ent.User, error) {
	r.Logger.Info().Int("user_id", user.ID).Msg("Attempting to update user password")
	var updatedUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		updatedUser, err = tx.User.
			UpdateOneID(user.ID).
			SetPassword(hashedPassword).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to update user password in DB")
			return errors.New("failed to update user")
		}
		r.Logger.Debug().Int("user_id", user.ID).Msg("User password updated in DB")

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for UpdateUsersPassword")
		return nil, err
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Info().Int("user_id", updatedUser.ID).Msg("User password updated successfully")
	return updatedUser, nil
}

func (r *UsersRepository) RemoveAccount(
	ctx context.Context,
	user *ent.User,
) error {
	r.Logger.Info().Int("user_id", user.ID).Msg("Attempting to soft delete user account")
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		if user, err := tx.User.
			UpdateOneID(user.ID).
			SetRemoved(true).
			Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Failed to mark user as removed in DB")
			err = errors.New("failed to remove user")
			return err
		}
		r.Logger.Debug().Int("user_id", user.ID).Msg("User account marked as removed in DB")

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for RemoveAccount")
		return errors.New("failed to remove user")
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Debug().Int("user_id", user.ID).Msg("User account successfully marked as removed")
	return err
}
