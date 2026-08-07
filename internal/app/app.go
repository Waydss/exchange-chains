package app

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

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
	transporthttp "github.com/L-ka080/DDB-6-case/internal/transport/http"
	"github.com/L-ka080/DDB-6-case/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg              *config.Config
	server           *httpserver.Server
	pool             *pgxpool.Pool
	redisConnection  *redisclient.Connection
	chainService     *cb.Service
	cleanupScheduler *rejection.Scheduler
}

type repositories struct {
	user     user.Repository
	category item.CategoryRepository
	item     item.Repository
	chain    cb.Repository
}

type services struct {
	user      *user.Service
	item      *item.Service
	chain     *cb.Service
	decline   *rejection.DeclineChainUseCase
	rejection rejection.Store
}

type handlers struct {
	user      *user.Handler
	item      *item.Handler
	chain     *cb.Handler
	rejection *rejection.Handler
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	pool, err := database.Connect(context.Background(), cfg.PG.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	redisConnection, err := redisclient.Connect(context.Background(), redisclient.Config{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}

	jwtProvider, err := security.NewJWT(cfg.JWT.Secret, cfg.JWT.TTL)
	if err != nil {
		closeDependencies(pool, redisConnection)
		return nil, fmt.Errorf("initialize JWT provider: %w", err)
	}

	fileStorage, err := storage.NewLocalStorage(cfg.Upload.Dir, cfg.Upload.MaxFileSize)
	if err != nil {
		closeDependencies(pool, redisConnection)
		return nil, fmt.Errorf("initialize file storage: %w", err)
	}

	repositories := newRepositories(pool)
	services := newServices(cfg, repositories, fileStorage, jwtProvider, redisConnection)
	handlers := newHandlers(cfg, services)

	cleanupScheduler, err := rejection.NewScheduler(services.rejection)
	if err != nil {
		closeDependencies(pool, redisConnection)
		return nil, fmt.Errorf("initialize rejected item cleanup: %w", err)
	}

	router := transporthttp.NewRouter(
		httpserver.ReadinessDependencies{
			Database: pool,
			Redis:    redisConnection,
		},
		cfg.Upload.Dir,
		transporthttp.Routes{
			Register:       handlers.user.Register,
			Login:          handlers.user.Login,
			Me:             handlers.user.Me,
			CreateItem:     handlers.item.Create,
			CreateChain:    handlers.chain.CreateChainsForItem,
			DeclineChain:   handlers.rejection.DeclineChain,
			AuthMiddleware: middleware.Auth(jwtProvider),
		},
	)

	return &App{
		cfg:              cfg,
		server:           httpserver.New(cfg.HTTP.Port, router),
		pool:             pool,
		redisConnection:  redisConnection,
		chainService:     services.chain,
		cleanupScheduler: cleanupScheduler,
	}, nil
}

func (a *App) Run() error {
	appContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer a.close()

	workersDone := a.chainService.StartChainRequestWorkers(appContext)
	cleanupDone := a.cleanupScheduler.Start(appContext)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- a.server.Run()
	}()

	log.Printf("API started on port %s in %s environment", a.cfg.HTTP.Port, a.cfg.App.Environment)

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("run HTTP server: %w", err)
		}
		return nil
	case <-appContext.Done():
		log.Print("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownContext); err != nil {
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

func newRepositories(pool *pgxpool.Pool) repositories {
	return repositories{
		user:     user.NewPostgresRepository(pool),
		category: category.NewPostgresRepository(pool),
		item:     item.NewPostgresRepository(pool),
		chain:    cb.NewPostgresRepository(pool),
	}
}

func newServices(
	cfg *config.Config,
	repositories repositories,
	fileStorage item.FileStorage,
	jwtProvider user.TokenProvider,
	redisConnection *redisclient.Connection,
) services {
	rejectionStore := rejection.NewRedisStore(redisConnection.Client(), cfg.Rejection.ItemTTL)

	return services{
		user:      user.NewService(repositories.user, security.NewPasswordHasher(), jwtProvider),
		item:      item.NewService(repositories.item, repositories.category, fileStorage, cfg.Upload.MaxItemPhotos),
		chain:     cb.NewService(repositories.chain),
		decline:   rejection.NewDeclineChainUseCase(rejectionStore),
		rejection: rejectionStore,
	}
}

func newHandlers(cfg *config.Config, services services) handlers {
	return handlers{
		user:      user.NewHandler(services.user),
		item:      item.NewHandler(services.item, cfg.Upload.MaxFileSize, cfg.Upload.MaxItemPhotos),
		chain:     cb.NewHandler(services.chain),
		rejection: rejection.NewHandler(services.decline),
	}
}

func (a *App) close() {
	closeDependencies(a.pool, a.redisConnection)
}

func closeDependencies(pool *pgxpool.Pool, redisConnection *redisclient.Connection) {
	if redisConnection != nil {
		if err := redisConnection.Close(); err != nil {
			log.Printf("close Redis connection: %v", err)
		}
	}
	if pool != nil {
		pool.Close()
	}
}
