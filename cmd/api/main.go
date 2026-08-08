// Command api starts the exchange-chain HTTP API.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/L-ka080/DDB-6-case/docs"
	"github.com/L-ka080/DDB-6-case/internal/category"
	cb "github.com/L-ka080/DDB-6-case/internal/chain_builder"
	"github.com/L-ka080/DDB-6-case/internal/config"
	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/L-ka080/DDB-6-case/internal/platform/database"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/L-ka080/DDB-6-case/internal/platform/redisclient"
	"github.com/L-ka080/DDB-6-case/internal/platform/security"
	"github.com/L-ka080/DDB-6-case/internal/platform/storage"
	"github.com/L-ka080/DDB-6-case/internal/rejection"
	"github.com/L-ka080/DDB-6-case/internal/user"
)

// @title Exchange Chain API
// @version 1.0
// @description Backend API проекта «Цепочка обмена».
//
// @contact.name Grigorev Vadim
// @contact.email grigorevvadim875@yandex.ru
//
// @host localhost:8080
// @BasePath /
// @schemes http
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите токен в формате: Bearer <access_token>
func main() {
	if err := run(); err != nil {
		log.Printf("application stopped with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	appContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(appContext, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	redisConnection, err := redisclient.Connect(appContext, redisclient.Config{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  cfg.RedisDialTimeout,
		ReadTimeout:  cfg.RedisReadTimeout,
		WriteTimeout: cfg.RedisWriteTimeout,
	})
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	defer func() {
		if err := redisConnection.Close(); err != nil {
			log.Printf("close Redis connection: %v", err)
		}
	}()

	jwtProvider, err := security.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("initialize JWT provider: %w", err)
	}

	userRepository := user.NewPostgresRepository(pool)
	userService := user.NewService(userRepository, security.NewPasswordHasher(), jwtProvider)
	userHandler := user.NewHandler(userService)
	categoryRepository := category.NewPostgresRepository(pool)
	itemRepository := item.NewPostgresRepository(pool)
	fileStorage, err := storage.NewLocalStorage(cfg.UploadDir, cfg.MaxUploadSize)
	if err != nil {
		return fmt.Errorf("initialize file storage: %w", err)
	}
	chainRepository := cb.NewPostgresRepository(pool)
	chainService := cb.NewService(chainRepository)
	chainHandler := cb.NewHandler(chainService)
	itemService := item.NewService(itemRepository, categoryRepository, fileStorage, cfg.MaxItemPhotos, chainService)
	itemHandler := item.NewHandler(itemService, cfg.MaxUploadSize, cfg.MaxItemPhotos)
	workersDone := chainService.StartChainRequestWorkers(appContext)
	rejectionStore := rejection.NewRedisStore(redisConnection.Client(), cfg.RejectedItemTTL)
	declineChainUseCase := rejection.NewDeclineChainUseCase(rejectionStore)
	declineChainHandler := rejection.NewHandler(declineChainUseCase)
	cleanupScheduler, err := rejection.NewScheduler(rejectionStore)
	if err != nil {
		return fmt.Errorf("initialize rejected item cleanup: %w", err)
	}
	cleanupDone := cleanupScheduler.Start(appContext)

	server := httpserver.New(cfg.Port, httpserver.ReadinessDependencies{
		Database: pool,
		Redis:    redisConnection,
	}, cfg.UploadDir, httpserver.Routes{
		Register:   userHandler.Register,
		Login:      userHandler.Login,
		Me:         userHandler.Me,
		CreateItem: itemHandler.Create,

		// FIXME Remove this at one point...
		CreateChain: chainHandler.CreateChainsForItem,

		GetChainsByItem: chainHandler.GetChainsByItem,
		GetChainsByUser: chainHandler.GetChainsByUser,
		DeclineChain:    declineChainHandler.DeclineChain,
		ListChains:      chainHandler.ListChains,
		AuthMiddleware:  middleware.Auth(jwtProvider),
	})
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Run()
	}()

	log.Printf("API started on port %s in %s environment", cfg.Port, cfg.Environment)

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("run HTTP server: %w", err)
		}
		return nil
	case <-appContext.Done():
		log.Print("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}

	if err := <-serverErrors; err != nil {
		return fmt.Errorf("stop HTTP server: %w", err)
	}

	<-workersDone
	<-cleanupDone

	log.Print("application stopped gracefully")
	return nil
}
