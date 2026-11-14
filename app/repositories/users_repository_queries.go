package repositories

import (
	"context"
	"errors"

	"users-service/app/models/ent"
	"users-service/app/models/ent/user"
)

func (r *UsersRepository) GetUserByID(ctx context.Context, ID int) (*ent.User, error) {
	r.Logger.Info().Int("user_id", ID).Msg("Attempting to get user by ID")
	var foundUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByID(ctx, tx, ID)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", ID).Msg("Failed to get user by ID within transaction")
			return err
		}

		r.Logger.Debug().Int("user_id", ID).Msg("User found by ID within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", ID).Msg("Transaction failed for GetUserByID")
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by ID")
	return foundUser, nil
}

func (r *UsersRepository) GetUserFunctionalDetailsByID(ctx context.Context, ID int) (*ent.User, error) {
	r.Logger.Info().Int("user_id", ID).Msg("Attempting to get user by ID")
	var foundUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserFunctionalByID(ctx, tx, ID)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", ID).Msg("Failed to get user by ID within transaction")
			return err
		}

		r.Logger.Debug().Int("user_id", ID).Msg("User found by ID within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", ID).Msg("Transaction failed for GetUserByID")
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by ID")
	return foundUser, nil
}

func (r *UsersRepository) GetUserFunctionalByID(ctx context.Context, ID int) (*ent.User, error) {
	r.Logger.Info().Int("user_id", ID).Msg("Attempting to get user by ID")
	var foundUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserFunctionalByID(ctx, tx, ID)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", ID).Msg("Failed to get user by ID within transaction")
			return err
		}

		r.Logger.Debug().Int("user_id", ID).Msg("User found by ID within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", ID).Msg("Transaction failed for GetUserByID")
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by ID")
	return foundUser, nil
}

func (r *UsersRepository) GetUserPublicByID(ctx context.Context, ID int) (*ent.User, error) {
	r.Logger.Info().Int("user_id", ID).Msg("Attempting to get user by ID")
	var foundUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserPublicByID(ctx, tx, ID)
		if err != nil {
			r.Logger.Error().Err(err).Int("user_id", ID).Msg("Failed to get user by ID within transaction")
			return err
		}

		r.Logger.Debug().Int("user_id", ID).Msg("User found by ID within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", ID).Msg("Transaction failed for GetUserByID")
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by ID")
	return foundUser, nil
}

func (r *UsersRepository) GetUserByMail(ctx context.Context, mail string) (*ent.User, error) {
	r.Logger.Info().Str("mail", mail).Msg("Attempting to get user by mail")
	var foundUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByMail(ctx, tx, mail)
		if err != nil {
			r.Logger.Error().Err(err).Str("mail", mail).Msg("Failed to get user by mail within transaction")
			return err
		}
		r.Logger.Debug().Str("mail", mail).Msg("User found by mail within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("mail", mail).Msg("Transaction failed for GetUserByMail")
		return nil, err
	}

	r.Logger.Info().Str("mail", foundUser.Mail).Msg("Successfully retrieved user by mail")
	return foundUser, nil
}

func (r *UsersRepository) GetUserByMailWithCodes(ctx context.Context, mail string) (*ent.User, error) {
	r.Logger.Info().Str("mail", mail).Msg("Attempting to get user by mail")
	var foundUser *ent.User
	var err error

	payload, _ := r.CacheStore.Get(ctx, "user:"+mail)
	if payload != nil {
		return r.CacheStore.DecacheUser(payload)
	}

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByMailWithCodes(ctx, tx, mail)
		if err != nil {
			r.Logger.Error().Err(err).Str("mail", mail).Msg("Failed to get user by mail within transaction")
			return err
		}
		r.Logger.Debug().Str("mail", mail).Msg("User found by mail within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("mail", mail).Msg("Transaction failed for GetUserByMail")
		return nil, err
	}

	r.Logger.Info().Str("mail", foundUser.Mail).Msg("Successfully retrieved user by mail")
	return foundUser, nil
}

func (r *UsersRepository) GetUserBySecondFactor(ctx context.Context, code string) (*ent.User, error) {
	r.Logger.Info().Str("second_factor_id", code).Msg("Attempting to get user by second factor code")
	var foundUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserBySecondFactor(ctx, tx, code)
		if err != nil {
			r.Logger.Error().Err(err).Str("second_factor_id", code).Msg("Failed to get user by second factor code within transaction")
			return err
		}

		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User found by second factor code within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("second_factor_id", code).Msg("Transaction failed for GetUserBySecondFactor")
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by second factor code")
	return foundUser, nil
}

func (r *UsersRepository) GetUserByVerificationCode(ctx context.Context, code string) (*ent.User, error) {
	r.Logger.Info().Str("verification_code", code).Msg("Attempting to get user by verification code")
	var foundUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByVerificationCode(ctx, tx, code)
		if err != nil {
			r.Logger.Error().Err(err).Str("verification_code", code).Msg("Failed to get user by verification code within transaction")
			return err
		}
		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User found by verification code within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("verification_code", code).Msg("Transaction failed for GetUserByVerificationCode")
		return nil, err
	}

	if err := UpdateCache(ctx, foundUser, r); err != nil {
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by verification code")
	return foundUser, nil
}

func (r *UsersRepository) GetUserByResetCode(ctx context.Context, code string) (*ent.User, error) {
	r.Logger.Info().Str("reset_code", code).Msg("Attempting to get user by reset code")
	var foundUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByResetCode(ctx, tx, code)
		if err != nil {
			r.Logger.Error().Err(err).Str("reset_code", code).Msg("Failed to get user by reset code within transaction")
			return err
		}
		r.Logger.Debug().Int("user_id", foundUser.ID).Msg("User found by reset code within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("reset_code", code).Msg("Transaction failed for GetUserByResetCode")
		return nil, err
	}

	if err := UpdateCache(ctx, foundUser, r); err != nil {
		return nil, err
	}

	r.Logger.Info().Int("user_id", foundUser.ID).Msg("Successfully retrieved user by reset code")
	return foundUser, nil
}

func (r *UsersRepository) GetUserByPhone(ctx context.Context, phone string) (*ent.User, error) {
	r.Logger.Info().Str("phone", phone).Msg("Attempting to get user by phone")
	var foundUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundUser, err = GetUserByPhone(ctx, tx, phone)
		if err != nil {
			r.Logger.Error().Err(err).Str("phone", phone).Msg("Failed to get user by phone within transaction")
			return err
		}
		r.Logger.Debug().Str("phone", phone).Msg("User found by phone within transaction")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Str("phone", phone).Msg("Transaction failed for GetUserByPhone")
		return nil, err
	}

	if _ = UpdateCache(ctx, foundUser, r); err != nil {
		return nil, err
	}

	r.Logger.Info().Str("phone", foundUser.Phone).Msg("Successfully retrieved user by phone")
	return foundUser, nil
}

func (r *UsersRepository) GetUsersPublic(ctx context.Context, page, pageSize int) ([]*ent.User, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	r.Logger.Debug().Int("page", page).Int("page_size", pageSize).Msg("Fetching public users with offset")

	foundUsers, err := r.DB.User.
		Query().
		Limit(pageSize).
		Offset(offset).
		Order(ent.Desc(user.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Database query failed for GetUsersPublic")
		return nil, errors.New("failed to fetch users public")
	}

	for _, user := range foundUsers {
		if err := UpdateCache(ctx, user, r); err != nil {
			r.Logger.Warn().Err(err).Int("user_id", user.ID).Msg("Failed to update cache for a user, continuing")
		}
	}

	return foundUsers, nil
}
