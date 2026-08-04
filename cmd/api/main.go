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
	"github.com/L-ka080/DDB-6-case/internal/config"
	"github.com/L-ka080/DDB-6-case/internal/platform/database"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver"
	"github.com/L-ka080/DDB-6-case/internal/platform/httpserver/middleware"
	"github.com/L-ka080/DDB-6-case/internal/platform/security"
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

	jwtProvider, err := security.NewJWT(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("initialize JWT provider: %w", err)
	}

	userRepository := user.NewPostgresRepository(pool)
	userService := user.NewService(userRepository, security.NewPasswordHasher(), jwtProvider)
	userHandler := user.NewHandler(userService)

	server := httpserver.New(cfg.Port, pool, httpserver.Routes{
		Register:       userHandler.Register,
		Login:          userHandler.Login,
		Me:             userHandler.Me,
		AuthMiddleware: middleware.Auth(jwtProvider),
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

	log.Print("application stopped gracefully")
	return nil
}
