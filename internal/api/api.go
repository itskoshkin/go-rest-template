package api

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	"go-rest-template/docs"
	"go-rest-template/internal/api/controllers"
	"go-rest-template/internal/api/middlewares"
	"go-rest-template/internal/config"
	"go-rest-template/internal/logger"
	"go-rest-template/internal/utils/gin"
	"go-rest-template/internal/utils/text"
	"go-rest-template/static"
)

type API struct {
	engine   *gin.Engine
	webCtrl  *controllers.WebController
	userCtrl *controllers.UserController
	itemCtrl *controllers.ItemController
}

func NewAPI(e *gin.Engine, wc *controllers.WebController, uc *controllers.UserController, ic *controllers.ItemController) *API {
	return &API{
		engine:   e,
		webCtrl:  wc,
		userCtrl: uc,
		itemCtrl: ic,
	}
}

func NewEngine() *gin.Engine {
	if viper.GetBool(config.GinReleaseMode) {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	_ = engine.SetTrustedProxies(nil) // Can nil produce an error? Or can a robot write a symphony?
	engine.HandleMethodNotAllowed = true

	return engine
}

func (api *API) LoadStaticFiles() {
	api.engine.SetHTMLTemplate(template.Must(template.New("").ParseFS(
		static.TemplatesFS,
		"templates/*.gohtml",
	)))
	api.engine.StaticFS("/static", http.FS(static.PublicFS))
}

func (api *API) RegisterMiddlewares() {
	api.engine.Use(middlewares.RequestID())
	api.engine.Use(ginutils.LoggingMiddlewares()...)
	api.engine.Use(middlewares.CORS())
}

func (api *API) RegisterRoutes() {
	// Web
	api.webCtrl.RegisterRoutes()

	// API
	api.userCtrl.RegisterRoutes()
	api.itemCtrl.RegisterRoutes()

	// Swagger
	docs.SwaggerInfo.Host = viper.GetString(config.WebAppDomain)
	docs.SwaggerInfo.BasePath = viper.GetString(config.ApiBasePath)
	api.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (api *API) Run() {
	fmt.Printf("Starting Gin engine...")

	server := &http.Server{Addr: fmt.Sprintf("%s:%s", viper.GetString(config.ApiHost), viper.GetString(config.ApiPort)), Handler: api.engine}

	errorChannel := make(chan error, 1)
	exitChannel := make(chan os.Signal, 1)
	signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(exitChannel)

	go func() {
		fmt.Println(text.Green("    Done."))
		logger.Info("Listening on %s...", fmt.Sprintf("%s:%s", viper.GetString(config.ApiHost), viper.GetString(config.ApiPort)))
		errorChannel <- server.ListenAndServe()
	}()

	select {
	case err := <-errorChannel:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server stopped unexpectedly: %v", err)
		}
		logger.Info("Server stopped.")
		return
	case sig := <-exitChannel:
		logger.Info("Received %s, shutting down...", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), viper.GetDuration(config.ApiShutdownTimeout))
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("Graceful shutdown failed: %v", err)
	}

	if err := <-errorChannel; err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("Server stopped unexpectedly: %v", err)
	}

	logger.Info("Server stopped.")
}
