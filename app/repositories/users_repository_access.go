package repositories

import (
	"context"
	"errors"

	"users-service/app/models/ent"
	"users-service/app/models/ent/user"
	"users-service/app/utils"
)

func (r *UsersRepository) AddSubscriptions(ctx context.Context, user *ent.User, subIDs []int) error {
	r.Logger.Debug().Int("user_id", user.ID).Msg("Attempting to add user's subscriptions")
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		toAppend := make([]int, 0)
		for _, newID := range subIDs {
			if !utils.ContainsInt(user.SubscriptionIds, newID) {
				toAppend = append(toAppend, newID)
			}
		}

		if len(toAppend) == 0 {
			return nil
		}

		newSubIDs := append(user.SubscriptionIds, toAppend...)

		if _, err := tx.User.
			UpdateOneID(user.ID).
			SetSubscriptionIds(newSubIDs).
			Save(ctx); err != nil {
			return errors.New("failed update user")
		}

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for AddSubscriptions")
		return errors.New("failed to add subscriptions")
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Debug().Int("user_id", user.ID).Msg("Successfully added subscriptions to user account")

	return nil
}

func (r *UsersRepository) RemoveSubscriptions(ctx context.Context, user *ent.User, subIDs []int) error {
	r.Logger.Debug().Int("user_id", user.ID).Msg("Attempting to revoke user's subscriptions")
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		newSubIDs := utils.RemoveIDs(user.SubscriptionIds, subIDs)

		if _, err := tx.User.
			UpdateOneID(user.ID).
			SetSubscriptionIds(newSubIDs).
			Save(ctx); err != nil {
			return errors.New("failed update user")
		}

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for RemoveSubscriptions")
		return errors.New("failed to remove subscriptions")
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Debug().Int("user_id", user.ID).Msg("Successfully removed subscriptions from user account")

	return nil
}

func (r *UsersRepository) CreateUserAction(
	ctx context.Context,
	user *ent.User,
	action, originDevice, details string,
) error {
	r.Logger.Debug().Int("user_id", user.ID).Msg("Attempting to create user action")
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		if _, err := tx.UserAction.
			Create().
			SetAction(action).
			SetDetails(details).
			AddAuthorIDs(user.ID).
			Save(ctx); err != nil {
			return errors.New("failed to create user action")
		}

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", user.ID).Msg("Transaction failed for CreateUserAction")
		return errors.New("failed to create user action")
	}

	InvalidateCache(ctx, user, r)

	r.Logger.Debug().Int("user_id", user.ID).Msg("Successfully created UserAction")
	return nil
}

func (r *UsersRepository) AddRole(ctx context.Context, userID, roleID int) (*ent.User, error) {
	r.Logger.Debug().Int("user_id", userID).Msg("Attempting to add role for user")
	var updatedUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		updatedUser, err = tx.User.
			UpdateOneID(userID).
			AddUserRoleIDs(roleID).
			Save(ctx)
		if err != nil {
			return errors.New("failed to add role for user")
		}

		updatedUser, _ = getUserWithRoles(ctx, tx, user.IDEQ(updatedUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", userID).Msg("Transaction failed for AddRole")
		return nil, errors.New("failed to add role for user")
	}

	InvalidateCache(ctx, updatedUser, r)

	r.Logger.Debug().Int("user_id", userID).Msg("Successfully added role for user")
	return updatedUser, nil
}

func (r *UsersRepository) RevokeRole(ctx context.Context, userID, roleID int) (*ent.User, error) {
	r.Logger.Debug().Int("user_id", userID).Msg("Attempting to revoke role for user")
	var updatedUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		updatedUser, err = tx.User.
			UpdateOneID(userID).
			RemoveUserRoleIDs(roleID).
			Save(ctx)
		if err != nil {
			return errors.New("failed to revoke role for user")
		}

		updatedUser, _ = getUserWithRoles(ctx, tx, user.IDEQ(updatedUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", userID).Msg("Transaction failed for RevokeRole")
		return nil, errors.New("failed to revoke role for user")
	}

	InvalidateCache(ctx, updatedUser, r)

	r.Logger.Debug().Int("user_id", userID).Msg("Successfully revoked role for user")
	return updatedUser, nil
}
