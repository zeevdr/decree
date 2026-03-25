package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	cfg := loadConfig()

	log.Printf("starting central-config-service on port %s", cfg.GRPCPort)
	log.Printf("enabled services: %s", strings.Join(cfg.EnableServices, ", "))

	// TODO: Initialize dependencies (DB, Redis, OTel)
	// TODO: Register gRPC services based on EnableServices
	// TODO: Start gRPC server

	log.Println("central-config-service stopped")
}

type config struct {
	GRPCPort       string
	DBWriteURL     string
	DBReadURL      string
	RedisURL       string
	EnableServices []string
	JWTIssuer      string
	JWTJWKSURL     string
	LogLevel       string
}

func loadConfig() config {
	enableServices := getEnv("ENABLE_SERVICES", "schema,config,audit")
	dbWriteURL := getEnv("DB_WRITE_URL", "")
	dbReadURL := getEnv("DB_READ_URL", dbWriteURL)

	return config{
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
				log.Fatalf("unknown service: %q (valid: schema, config, audit)", svc)
			}
		}
	}
	if len(services) == 0 {
		log.Fatal("no services enabled")
	}
	return services
}

func init() {
	_ = fmt.Sprintf // Avoid unused import — will be removed when wiring is added.
}
