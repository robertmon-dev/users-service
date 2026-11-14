package repositories

import (
	"context"

	"users-service/app/infrastructure"
	"users-service/app/models/ent"
	_ "users-service/app/models/ent/runtime"
	"users-service/app/models/requests"

	"entgo.io/ent/dialect/sql"
	"github.com/rs/zerolog"
)

type UsersRepository struct {
	CacheStore infrastructure.CacheStoreInterface
	DB         *ent.Client
	Driver     *sql.Driver
	Logger     zerolog.Logger
}

type UsersRepositoryInterface interface {
	AddRole(ctx context.Context, userID, roleID int) (*ent.User, error)
	RevokeRole(ctx context.Context, userID, roleID int) (*ent.User, error)

	CreateUser(ctx context.Context, req *requests.Register, hashedPassword string) (*ent.User, *ent.ActivationCode, error)
	CreateResetCode(ctx context.Context, user *ent.User) (*ent.ResetCode, error)
	CreateSecondFactorCode(ctx context.Context, user *ent.User, device *ent.UserDevice) (*ent.SecondFactorCode, error)
	CreateUserAction(ctx context.Context, user *ent.User, action, originDevice, details string) error

	GetUserByID(ctx context.Context, ID int) (*ent.User, error)
	GetUserFunctionalByID(ctx context.Context, ID int) (*ent.User, error)
	GetUserFunctionalDetailsByID(ctx context.Context, ID int) (*ent.User, error)
	GetUserPublicByID(ctx context.Context, ID int) (*ent.User, error)
	GetUserByMail(ctx context.Context, mail string) (*ent.User, error)
	GetUserByMailWithCodes(ctx context.Context, mail string) (*ent.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*ent.User, error)
	GetUserBySecondFactor(ctx context.Context, code string) (*ent.User, error)
	GetUserByVerificationCode(ctx context.Context, code string) (*ent.User, error)
	GetUserByResetCode(ctx context.Context, code string) (*ent.User, error)
	GetUsersPublic(ctx context.Context, page, pageSize int) ([]*ent.User, error)

	FindOrCreateDevice(ctx context.Context, userID int, req *requests.Device) (*ent.UserDevice, error)

	ActivateAccount(ctx context.Context, code string) (*ent.User, error)
	AddSubscriptions(ctx context.Context, user *ent.User, subIDs []int) error
	VerifyAccount(ctx context.Context, code string) (*ent.User, error)
	UpdateUser(ctx context.Context, user *ent.User, req *requests.User) (*ent.User, error)
	UpdateUsersPassword(ctx context.Context, user *ent.User, hashedPassword string) (*ent.User, error)
	UpdateUsersDetails(ctx context.Context, user *ent.User, req *requests.Details) (*ent.User, error)
	UpdateUsersSettings(ctx context.Context, user *ent.User, req *requests.Settings) (*ent.User, error)
	UpdateEntrepreneurDetails(ctx context.Context, user *ent.User, req *requests.EntrepreneurDetails) (*ent.User, error)
	UpdateLocation(ctx context.Context, user *ent.User, req *requests.Location) (*ent.User, error)

	RemoveResetCode(ctx context.Context, user *ent.User) (*ent.User, error)
	RemoveSecondFactorCode(ctx context.Context, user *ent.User) (*ent.User, error)
	RemoveAccount(ctx context.Context, user *ent.User) error
	RemoveSubscriptions(ctx context.Context, user *ent.User, subIDs []int) error

	Ping() error
}

var _ UsersRepositoryInterface = (*UsersRepository)(nil)

func NewUsersRepository(
	cacheStore infrastructure.CacheStoreInterface,
	entClient *ent.Client,
	driver *sql.Driver,
	logger zerolog.Logger,
) *UsersRepository {
	return &UsersRepository{
		CacheStore: cacheStore,
		DB:         entClient,
		Driver:     driver,
		Logger:     logger,
	}
}

func (r *UsersRepository) Ping() error {
	r.Logger.Debug().Msg("Attempting to ping database")

	err := r.Driver.DB().Ping()
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to ping database")
		return err
	}

	r.Logger.Debug().Msg("Database ping successful")
	return nil
}
