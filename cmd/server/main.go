package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/redis/go-redis/v9"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
	"github.com/zeevdr/central-config-service/internal/audit"
	"github.com/zeevdr/central-config-service/internal/auth"
	"github.com/zeevdr/central-config-service/internal/cache"
	"github.com/zeevdr/central-config-service/internal/config"
	"github.com/zeevdr/central-config-service/internal/pubsub"
	"github.com/zeevdr/central-config-service/internal/schema"
	"github.com/zeevdr/central-config-service/internal/server"
	"github.com/zeevdr/central-config-service/internal/storage"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg := loadConfig()
	logger := newLogger(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database.
	db, err := storage.NewDB(ctx, cfg.DBWriteURL, cfg.DBReadURL)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect to database", "error", err)
		return 1
	}
	defer db.Close()
	logger.InfoContext(ctx, "connected to database")

	// Redis.
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse redis url", "error", err)
		return 1
	}
	redisClient := redis.NewClient(redisOpts)
	defer func() { _ = redisClient.Close() }()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.ErrorContext(ctx, "failed to connect to redis", "error", err)
		return 1
	}
	logger.InfoContext(ctx, "connected to redis")

	// Cache and pub/sub.
	configCache := cache.NewRedisCache(redisClient)
	publisher := pubsub.NewRedisPublisher(redisClient)
	subscriber := pubsub.NewRedisSubscriber(redisClient, logger)
	defer func() { _ = publisher.Close() }()
	defer func() { _ = subscriber.Close() }()

	// Auth interceptor.
	var authInterceptor *auth.Interceptor
	if cfg.JWTJWKSURL != "" {
		authInterceptor, err = auth.NewInterceptor(ctx, cfg.JWTJWKSURL, cfg.JWTIssuer, logger)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create auth interceptor", "error", err)
			return 1
		}
		defer authInterceptor.Close()
		logger.InfoContext(ctx, "JWT auth enabled", "jwks_url", cfg.JWTJWKSURL)
	} else {
		logger.WarnContext(ctx, "JWT auth disabled — no JWT_JWKS_URL configured")
	}

	// gRPC server.
	srv, err := server.New(server.Config{
		GRPCPort:        cfg.GRPCPort,
		EnableServices:  cfg.EnableServices,
		Logger:          logger,
		AuthInterceptor: authInterceptor,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to create server", "error", err)
		return 1
	}

	// Register services.
	if srv.IsServiceEnabled("schema") {
		schemaStore := schema.NewPGStore(db.WritePool, db.ReadPool)
		schemaSvc := schema.NewService(schemaStore, logger)
		pb.RegisterSchemaServiceServer(srv.GRPCServer(), schemaSvc)
		srv.SetServiceHealthy("centralconfig.v1.SchemaService")
		logger.InfoContext(ctx, "schema service enabled")
	}
	if srv.IsServiceEnabled("config") {
		configStore := config.NewPGStore(db.WritePool, db.ReadPool)
		configSvc := config.NewService(configStore, configCache, publisher, subscriber, logger)
		pb.RegisterConfigServiceServer(srv.GRPCServer(), configSvc)
		srv.SetServiceHealthy("centralconfig.v1.ConfigService")
		logger.InfoContext(ctx, "config service enabled")
	}
	if srv.IsServiceEnabled("audit") {
		auditStore := audit.NewPGStore(db.WritePool, db.ReadPool)
		auditSvc := audit.NewService(auditStore, logger)
		pb.RegisterAuditServiceServer(srv.GRPCServer(), auditSvc)
		srv.SetServiceHealthy("centralconfig.v1.AuditService")
		logger.InfoContext(ctx, "audit service enabled")
	}

	// Start server in background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.InfoContext(ctx, "received signal, shutting down", "signal", sig)
	case err := <-errCh:
		logger.ErrorContext(ctx, "server error", "error", err)
	}

	cancel()
	srv.GracefulStop(ctx)
	logger.InfoContext(ctx, "central-config-service stopped")
	return 0
}

type serverConfig struct {
	GRPCPort       string
	DBWriteURL     string
	DBReadURL      string
	RedisURL       string
	EnableServices []string
	JWTIssuer      string
	JWTJWKSURL     string
	LogLevel       string
}

func loadConfig() serverConfig {
	enableServices := getEnv("ENABLE_SERVICES", "schema,config,audit")
	dbWriteURL := getEnv("DB_WRITE_URL", "")
	dbReadURL := getEnv("DB_READ_URL", dbWriteURL)

	return serverConfig{
		GRPCPort:       getEnv("GRPC_PORT", "9090"),
		DBWriteURL:     dbWriteURL,
		DBReadURL:      dbReadURL,
		RedisURL:       getEnv("REDIS_URL", ""),
		EnableServices: parseServices(enableServices),
		JWTIssuer:      getEnv("JWT_ISSUER", ""),
		JWTJWKSURL:     getEnv("JWT_JWKS_URL", ""),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseServices(s string) []string {
	var services []string
	for _, svc := range strings.Split(s, ",") {
		svc = strings.TrimSpace(svc)
		if svc != "" {
			switch svc {
			case "schema", "config", "audit":
				services = append(services, svc)
			default:
				slog.Error("unknown service", "service", svc)
				os.Exit(1)
			}
		}
	}
	if len(services) == 0 {
		slog.Error("no services enabled")
		os.Exit(1)
	}
	return services
}

func newLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
}
