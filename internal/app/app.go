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
	transporthttp "github.com/L-ka080/DDB-6-case/internal/transport/http"
	"github.com/L-ka080/DDB-6-case/internal/user"
	"github.com/L-ka080/DDB-6-case/internal/wish"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg              *config.Config
	server           *httpserver.Server
	pool             *pgxpool.Pool
	redisConnection  *redisclient.Connection
	chainService     *cb.Service
	cleanupScheduler *cb.Scheduler
}

type repositories struct {
	user     user.Repository
	category item.CategoryRepository
	item     item.Repository
	wish     wish.Repository
	chain    cb.Repository
}

type services struct {
	user           *user.Service
	item           *item.Service
	wish           *wish.Service
	chain          *cb.Service
	rejectionStore cb.RejectionStore
}

type handlers struct {
	user  *user.Handler
	item  *item.Handler
	wish  *wish.Handler
	chain *cb.Handler
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	setupContext := context.Background()
	pool, err := database.Connect(setupContext, cfg.PG.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	redisConnection, err := redisclient.Connect(setupContext, redisclient.Config{
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

	fileStorage, err := newFileStorage(setupContext, cfg)
	if err != nil {
		closeDependencies(pool, redisConnection)
		return nil, fmt.Errorf("initialize file storage: %w", err)
	}

	repositories := newRepositories(pool)
	services := newServices(cfg, repositories, fileStorage, jwtProvider, redisConnection)
	handlers := newHandlers(cfg, services)

	cleanupScheduler, err := cb.NewScheduler(services.rejectionStore)
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
			Register:   handlers.user.Register,
			Login:      handlers.user.Login,
			Me:         handlers.user.Me,
			CreateItem: handlers.item.Create,
			ListItems:  handlers.item.List,
			GetItem:    handlers.item.GetByID,
			ListWishes: handlers.wish.List,

			// FIXME Remove this at one point...
			CreateChain: handlers.chain.CreateChainsForItem,

			GetChainsByUser: handlers.chain.GetChainsByUser,
			GetChainsByItem: handlers.chain.GetChainsByItem,

			UpdatePublicRating: handlers.user.UpdatePublicRating,
			GetUserRatings:     handlers.user.GetUserRatings,

			GetChainByID:   handlers.chain.GetChainByID,
			AcceptChain:    handlers.chain.AcceptChain,
			DeclineChain:   handlers.chain.DeclineChain,
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

func newFileStorage(ctx context.Context, cfg *config.Config) (item.FileStorage, error) {
	switch cfg.Storage.Driver {
	case "local":
		return storage.NewLocalStorage(cfg.Upload.Dir, cfg.Upload.MaxFileSize)
	case "s3":
		return storage.NewS3Storage(ctx, storage.S3Config{
			Endpoint:      cfg.S3.Endpoint,
			AccessKey:     cfg.S3.AccessKey,
			SecretKey:     cfg.S3.SecretKey,
			Bucket:         cfg.S3.Bucket,
			Region:         cfg.S3.Region,
			PublicBaseURL:  cfg.S3.PublicBaseURL,
			UseSSL:         cfg.S3.UseSSL,
			MaxFileSize:    cfg.Upload.MaxFileSize,
			EnsureBucket:   cfg.S3.EnsureBucket,
		})
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.Storage.Driver)
	}
}

func newRepositories(pool *pgxpool.Pool) repositories {
	return repositories{
		user:     user.NewPostgresRepository(pool),
		category: category.NewPostgresRepository(pool),
		item:     item.NewPostgresRepository(pool),
		wish:     wish.NewPostgresRepository(pool),
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
	rejectionStore := cb.NewRedisStore(redisConnection.Client(), cfg.Rejection.ItemTTL)

	userService := user.NewService(repositories.user, security.NewPasswordHasher(), jwtProvider)

	chainService := cb.NewService(repositories.chain, userService, rejectionStore)
	itemService := item.NewService(repositories.item, repositories.category, fileStorage, cfg.Upload.MaxItemPhotos, chainService)
	wishService := wish.NewService(repositories.wish)

	return services{
		user:           userService,
		item:           itemService,
		wish:           wishService,
		chain:          chainService,
		rejectionStore: rejectionStore,
	}
}

func newHandlers(cfg *config.Config, services services) handlers {
	return handlers{
		user:  user.NewHandler(services.user),
		item:  item.NewHandler(services.item, cfg.Upload.MaxFileSize, cfg.Upload.MaxItemPhotos),
		wish:  wish.NewHandler(services.wish),
		chain: cb.NewHandler(services.chain),
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
