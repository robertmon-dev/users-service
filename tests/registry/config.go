package registry

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"log"
	"time"

	"users-service/app/config"
	"users-service/app/controllers"
	"users-service/app/infrastructure"
	"users-service/app/middlewares"
	"users-service/app/models/ent"
	"users-service/app/repositories"
	"users-service/app/routes"
	"users-service/app/services"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func NewTestAppContainer(
	testSettings *config.Settings,
	db *ent.Client,
	redis *redis.Client,
	kafkaProducer *kafka.Producer,
	logger zerolog.Logger,
	pubKey crypto.PublicKey,
	privKey crypto.PrivateKey,
) *AppContainer {
	cache := infrastructure.NewCacheStore(redis, logger, testSettings.EncryptionSecretKey)
	repo := repositories.NewUsersRepository(cache, db, nil, logger)
	notifier := infrastructure.NewEventNotifier(kafkaProducer, logger, "notifications", time.Second*10)
	consumerManager := infrastructure.NewConsumerManager(logger)

	privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		log.Fatalf("failed to marshal private key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		log.Fatalf("failed to marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	tokenDuration, err := time.ParseDuration(testSettings.TokenDuration)
	if err != nil {
		log.Fatalf("failed to parse token duration: %v", err)
	}

	bearerService := services.NewBearerService(logger, string(pubPEM), string(privPEM), tokenDuration)
	passwordService := services.NewPasswordService(logger, []byte(testSettings.HashingPepper), 10)

	serviceContainer := &ServiceContainer{
		AuthService:        services.NewAuthService(repo, notifier, logger, bearerService, passwordService),
		UsersService:       services.NewUsersService(repo, notifier, logger),
		DiagnosticsService: services.NewDiagnosticsService(repo, cache, notifier, consumerManager, logger),
	}

	controllerContainer := &ControllerContainer{
		AuthController: controllers.NewAuthController(
			serviceContainer.AuthService,
			serviceContainer.UsersService,
			logger,
		),
		UsersController: controllers.NewUsersController(
			serviceContainer.UsersService,
			logger,
		),
		DiagnosticsController: controllers.NewDiagnosticsController(
			serviceContainer.DiagnosticsService,
			logger,
		),
	}

	router := gin.New()
	RegisterAllRoutes(router, controllerContainer, serviceContainer, cache, logger, testSettings)

	return &AppContainer{
		Router:      router,
		Services:    serviceContainer,
		Controllers: controllerContainer,
		Functions:   &FunctionRegistry,
	}
}

func RegisterAllRoutes(
	router *gin.Engine,
	controllerContainer *ControllerContainer,
	serviceContainer *ServiceContainer,
	cache infrastructure.CacheStoreInterface,
	logger zerolog.Logger,
	testSettings *config.Settings,
) {
	middlewaresStore := middlewares.NewMiddlewares(cache, logger, testSettings.ApiVersion, testSettings.Environment)
	tracker := middlewares.NewRequestTracker(40)

	router.Use(middlewaresStore.RequestAuthenticationMiddleware())

	api := router.Group("/api/v1")
	routes.RegisterAuthRoutes("/auth", api, controllerContainer.AuthController, tracker, middlewaresStore, logger)
	routes.RegisterUserRoutes("/users", api, controllerContainer.UsersController, serviceContainer.AuthService, tracker, middlewaresStore, logger)
	routes.RegisterDiagnosticsRoutes("/diagnostics", api, controllerContainer.DiagnosticsController, tracker, middlewaresStore, logger)
}
