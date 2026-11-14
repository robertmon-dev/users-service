package repositories

import (
	"context"
	"errors"
	"strconv"

	"users-service/app/models/ent"
	"users-service/app/models/ent/entrepreneurdetails"
	"users-service/app/models/ent/location"
	"users-service/app/models/ent/user"
	"users-service/app/models/ent/userdetails"
	"users-service/app/models/ent/userdevice"
	"users-service/app/models/requests"
)

func (r *UsersRepository) UpdateUser(ctx context.Context, targetUser *ent.User, req *requests.User) (*ent.User, error) {
	r.Logger.Info().Int("user_id", targetUser.ID).Msg("Attempting to update user mail/phone")
	var updatedUser *ent.User

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		userUpdater := tx.User.UpdateOneID(targetUser.ID)
		updated := false

		if req.Phone != targetUser.Phone {
			if exists, _ := tx.User.
				Query().
				Where(user.PhoneEQ(req.Phone)).
				Exist(ctx); exists {
				return errors.New("phone already exists")
			}

			userUpdater.SetPhone(req.Phone)
			updated = true
			r.Logger.Debug().Int("user_id", targetUser.ID).Str("old_phone", targetUser.Phone).Str("new_phone", req.Phone).Msg("Updating user phone")
		}

		if req.Mail != targetUser.Mail {
			if exists, _ := tx.User.
				Query().
				Where(user.MailEQ(req.Mail)).
				Exist(ctx); exists {
				return errors.New("mail already exists")
			}

			userUpdater.SetMail(req.Mail)
			updated = true
			r.Logger.Debug().Int("user_id", targetUser.ID).Str("old_mail", targetUser.Mail).Str("new_mail", req.Mail).Msg("Updating user mail")
		}

		if updated {
			if _, err := userUpdater.Save(ctx); err != nil {
				r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Failed to update user mail/phone in DB")
				return errors.New("failed to update user's mail/phone")
			}

			r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User mail/phone updated in DB")
		} else {
			r.Logger.Debug().Int("user_id", targetUser.ID).Msg("No changes detected for user mail/phone, skipping update")
		}

		updatedUser, _ = getUserFunctionalWithDetails(ctx, tx, user.IDEQ(targetUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Transaction failed for UpdateUser")
		return nil, errors.New("failed to update user")
	}

	InvalidateCache(ctx, targetUser, r)

	r.Logger.Info().Int("user_id", updatedUser.ID).Msg("User mail/phone updated successfully")
	return updatedUser, nil
}

func (r *UsersRepository) UpdateUsersDetails(
	ctx context.Context,
	targetUser *ent.User,
	req *requests.Details,
) (*ent.User, error) {
	r.Logger.Debug().Int("user_id", targetUser.ID).Msg("Attempting to update user details")
	var updatedUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		detailsUpdated := tx.UserDetails.
			UpdateOneID(targetUser.Edges.UserDetails.ID).
			SetFirstName(req.FirstName).
			SetLastName(req.LastName)

		if _, err = detailsUpdated.Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Failed to update user's details in DB")
			return errors.New("failed to update user's details")
		}
		r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User details updated in DB")

		updatedUser, err = getUserFunctionalWithDetails(ctx, tx, user.IDEQ(targetUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Transaction failed for UpdateUsersDetails")
		return nil, errors.New("failed to update user")
	}

	InvalidateCache(ctx, targetUser, r)

	r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User details updated successfully")
	return updatedUser, nil
}

func (r *UsersRepository) UpdateUsersSettings(ctx context.Context, targetUser *ent.User, req *requests.Settings) (*ent.User, error) {
	r.Logger.Debug().Int("user_id", targetUser.ID).Msg("Attempting to update user settings")
	var updatedUser *ent.User
	var err error

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		settingsUpdater := tx.UserSettings.
			UpdateOneID(targetUser.Edges.UserSettings.ID).
			SetNightMode(req.NightMode).
			SetTwoFactor(req.TwoFactor).
			SetSecondFactorTargetID(req.SecondFactorTargetID)

		r.Logger.Debug().Int("user_id", targetUser.ID).
			Bool("night_mode", req.NightMode).
			Bool("two_factor", req.TwoFactor).
			Int("second_factor_target_id", req.SecondFactorTargetID).
			Msg("Updating user settings fields")

		settingsUpdater.ClearNotificationTargetDevices().
			AddNotificationTargetDeviceIDs(req.NotificationsTargetDeviceIDs...)
		r.Logger.Debug().Int("user_id", targetUser.ID).
			Ints("notification_target_device_ids", req.NotificationsTargetDeviceIDs).
			Msg("Updating notification target devices")

		if _, err = settingsUpdater.Save(ctx); err != nil {
			r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Failed to update user's settings in DB")
			return errors.New("failed to update user's settings")
		}
		r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User settings updated in DB")

		updatedUser, _ = getUserWithSettings(ctx, tx, user.IDEQ(targetUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Transaction failed for UpdateUsersSettings")
		return nil, errors.New("failed to update user")
	}

	InvalidateCache(ctx, targetUser, r)

	r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User settings updated successfully")
	return updatedUser, nil
}

func (r *UsersRepository) FindOrCreateDevice(
	ctx context.Context,
	userID int,
	req *requests.Device,
) (*ent.UserDevice, error) {
	r.Logger.Debug().Int("user_id", userID).Str("device_token", req.DeviceToken).Msg("Attempting to find or create device")
	var device *ent.UserDevice

	if err := WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		d, err := tx.UserDevice.Query().
			Where(
				userdevice.HasOwnerWith(user.IDEQ(userID)),
				userdevice.Token(req.DeviceToken),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				r.Logger.Debug().Int("user_id", userID).Str("device_token", req.DeviceToken).Msg("Device not found, attempting to create new device")
				newDevice, err := tx.UserDevice.
					Create().
					SetOwnerID(userID).
					SetToken(req.DeviceToken).
					SetIPAddress(req.IPAddress).
					SetBrowserName(req.BrowserName).
					SetBrowserVersion(req.BrowserVersion).
					SetOsName(req.OSName).
					SetOsVersion(req.OSVersion).
					SetUserAgent(req.UserAgent).
					Save(ctx)
				if err != nil {
					r.Logger.Error().Err(err).Int("user_id", userID).Str("device_token", req.DeviceToken).Msg("Failed to create new device")
					return err
				}
				device = newDevice
				r.Logger.Debug().Int("user_id", userID).Int("device_id", device.ID).Msg("New device created successfully")

				user, _ := GetUserPublicByID(ctx, tx, userID)
				InvalidateCache(ctx, user, r)

				return nil
			}
			r.Logger.Error().Err(err).Int("user_id", userID).Str("device_token", req.DeviceToken).Msg("Failed to query device in DB")
			return err
		}

		device = d
		r.Logger.Debug().Int("user_id", userID).Int("device_id", device.ID).Msg("Existing device found")
		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", userID).Str("device_token", req.DeviceToken).Msg("Transaction failed for FindOrCreateDevice")
		return nil, errors.New("failed to find or create device in transaction")
	}

	r.Logger.Debug().Int("user_id", userID).Int("device_id", device.ID).Msg("Device operation completed successfully")
	return device, nil
}

func (r *UsersRepository) UpdateEntrepreneurDetails(
	ctx context.Context,
	targetUser *ent.User,
	req *requests.EntrepreneurDetails,
) (*ent.User, error) {
	r.Logger.Info().Int("user_id", targetUser.ID).Msg("Attempting to update entrepreneur details")
	var updatedUser *ent.User
	var err error

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		details, _ := tx.EntrepreneurDetails.
			Query().
			Where(entrepreneurdetails.HasUserDetailsWith(userdetails.IDEQ(targetUser.Edges.UserDetails.ID))).
			Only(ctx)
		if details == nil {
			if _, err := tx.EntrepreneurDetails.
				Create().
				SetUserDetailsID(targetUser.Edges.UserDetails.ID).
				SetBusinessName(req.BusinessName).
				SetNip(req.NIP).
				SetKrs(req.KRS).
				SetDescription(req.Description).
				SetIncome(req.Income).
				SetCosts(req.Costs).
				SetFundingCapital(req.FundingCapital).
				SetManagementCouncilMembers(req.ManagementCouncilMembers).
				SetDecisionMakers(req.DecisionMakers).
				SetBusinessPhoneNumber(req.BusinessPhoneNumber).
				SetBusinessMail(req.BusinessMail).
				SetWebsiteAddress(req.WebsiteAddress).
				Save(ctx); err != nil {
				return errors.New("failed to create entrepreneur details")
			}
		} else {
			if _, err := tx.EntrepreneurDetails.
				UpdateOneID(details.ID).
				SetUserDetailsID(targetUser.Edges.UserDetails.ID).
				SetBusinessName(req.BusinessName).
				SetNip(req.NIP).
				SetKrs(req.KRS).
				SetDescription(req.Description).
				SetIncome(req.Income).
				SetCosts(req.Costs).
				SetFundingCapital(req.FundingCapital).
				SetManagementCouncilMembers(req.ManagementCouncilMembers).
				SetDecisionMakers(req.DecisionMakers).
				SetBusinessPhoneNumber(req.BusinessPhoneNumber).
				SetBusinessMail(req.BusinessMail).
				SetWebsiteAddress(req.WebsiteAddress).
				Save(ctx); err != nil {
				return errors.New("failed to update entrepreneur details")
			}
		}

		updatedUser, err = getUserFunctionalWithDetails(ctx, tx, user.IDEQ(targetUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Transaction failed for UpdateEntrepreneurDetails")
		return nil, errors.New("failed to update entrepreneur details")
	}

	InvalidateCache(ctx, targetUser, r)

	return updatedUser, nil
}

func (r *UsersRepository) UpdateLocation(
	ctx context.Context,
	targetUser *ent.User,
	req *requests.Location,
) (*ent.User, error) {
	r.Logger.Debug().Int("user_id", targetUser.ID).Msg("Attempting to update user location")
	var updatedUser *ent.User
	var err error

	buildingNumber, err := strconv.Atoi(req.BuildingNumber)
	if err != nil {
		return nil, errors.New("building number must be numeric")
	}

	apartmentNumber, err := strconv.Atoi(req.ApartmentNumber)
	if err != nil {
		return nil, errors.New("apartment number must be numeric")
	}

	if err = WithTransaction(ctx, r.DB, func(tx *ent.Tx) error {
		foundLocation, _ := tx.Location.
			Query().
			Where(location.HasUserDetailsWith(userdetails.IDEQ(targetUser.Edges.UserDetails.ID))).
			Only(ctx)
		if foundLocation != nil {
			if _, err := tx.Location.
				UpdateOneID(foundLocation.ID).
				SetCountry(req.Country).
				SetProvince(req.Province).
				SetCity(req.City).
				SetPostalCode(req.PostalCode).
				SetStreet(req.Street).
				SetBuildingNumber(buildingNumber).
				SetApartmentNumber(apartmentNumber).
				Save(ctx); err != nil {
				return errors.New("failed to update user's location")
			}
		} else {
			if _, err := tx.Location.
				Create().
				SetUserDetailsID(targetUser.Edges.UserDetails.ID).
				SetCountry(req.Country).
				SetProvince(req.Province).
				SetCity(req.City).
				SetPostalCode(req.PostalCode).
				SetStreet(req.Street).
				SetBuildingNumber(buildingNumber).
				SetApartmentNumber(apartmentNumber).
				Save(ctx); err != nil {
				return errors.New("failed to create new location")
			}
		}

		r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User location updated in DB")

		updatedUser, _ = getUserFunctionalWithDetails(ctx, tx, user.IDEQ(targetUser.ID))

		return nil
	}); err != nil {
		r.Logger.Error().Err(err).Int("user_id", targetUser.ID).Msg("Transaction failed for UpdateLocation")
		return nil, errors.New("failed to update location")
	}

	InvalidateCache(ctx, targetUser, r)

	r.Logger.Debug().Int("user_id", targetUser.ID).Msg("User location successfully updated")

	return updatedUser, nil
}
