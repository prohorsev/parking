package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr      string
	DatabaseURL     string
	MigrationsPath  string
	RabbitMQURL     string
	ProviderAPort   int
	ProviderBPort   int
	RefreshInterval time.Duration
}

func Load() Config {
	return Config{
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://parking:parking@localhost:5432/parking?sslmode=disable"),
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "./migrations"),
		RabbitMQURL:     getEnv("RABBITMQ_URL", "amqp://parking:parking@localhost:5672/"),
		ProviderAPort:   getEnvInt("PROVIDER_A_PORT", 9001),
		ProviderBPort:   getEnvInt("PROVIDER_B_PORT", 9002),
		RefreshInterval: getEnvDuration("REFRESH_INTERVAL", 60*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
