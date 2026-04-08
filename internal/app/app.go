package app

import (
	"context"
	"time"

	"go-rest-template/internal/api"
	"go-rest-template/internal/api/controllers"
	"go-rest-template/internal/api/middlewares"
	"go-rest-template/internal/config"
	"go-rest-template/internal/logger"
	"go-rest-template/internal/repository/cache"
	"go-rest-template/internal/repository/storage"
	"go-rest-template/internal/services"
	"go-rest-template/pkg/minio"
	"go-rest-template/pkg/postgres"
	"go-rest-template/pkg/redis"
)

type App struct {
	API *api.API
}

func New() *App {
	config.LoadConfig()
	logger.SetupLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Databases/clients
	db, err := postgres.NewInstance(ctx, config.PostgresConfig())
	if err != nil {
		logger.Fatal(err)
	}
	rc, err := redis.NewClient(ctx, config.RedisConfig())
	if err != nil {
		logger.Fatal(err)
	}
	s3, err := minio.NewClient(ctx, config.MinioConfig())
	if err != nil {
		logger.Fatal(err)
	}

	// Storages
	userStorage := storage.NewUserStorage(db)
	itemStorage := storage.NewItemStorage(db)
	tokenStorage := cache.NewTokenStorage(rc)
	objStorage := storage.NewMinioStorage(s3)

	// Services
	authSvc := services.NewAuthService(tokenStorage)
	emailSvc := services.NewEmailService()
	userSvc := services.NewUserService(userStorage, tokenStorage, emailSvc)
	itemSvc := services.NewItemService(itemStorage)
	imgSvc := services.NewImageService(objStorage)

	// API
	e := api.NewEngine()
	mw := middlewares.NewMiddlewares(authSvc)

	// Controllers
	webCtrl := controllers.NewWebController(e, userSvc)
	userCtrl := controllers.NewUserController(e, mw, authSvc, userSvc, imgSvc)
	itemCtrl := controllers.NewItemController(e, mw, itemSvc, imgSvc)

	return &App{API: api.NewAPI(e, webCtrl, userCtrl, itemCtrl)}
}

func (a *App) Run() {
	a.API.LoadStaticFiles()
	a.API.RegisterMiddlewares()
	a.API.RegisterRoutes()
	a.API.Run()
}
