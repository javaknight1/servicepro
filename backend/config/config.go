package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Auth     AuthConfig
	AWS      AWSConfig
}

// AWSConfig holds AWS configuration
type AWSConfig struct {
	Region          string
	SESFromEmail    string
	S3Bucket        string
	AccessKeyID     string
	SecretAccessKey string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port        string
	Env         string
	FrontendURL string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret            string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	MaxLoginAttempts  int
	LockoutDuration   time.Duration
	RateLimitWindow   time.Duration
	RateLimitAttempts int
}

// Load reads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Env:         getEnv("ENV", "development"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgresql://postgres:password@localhost:5432/servicepro?sslmode=disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
			AccessExpiration:  time.Hour,          // 1 hour as per requirements
			RefreshExpiration: time.Hour * 24 * 7, // 7 days as per requirements
		},
		Auth: AuthConfig{
			MaxLoginAttempts:  5,                // 5 failed attempts before lockout
			LockoutDuration:   time.Minute * 30, // 30 minutes lockout
			RateLimitWindow:   time.Minute * 15, // 15 minutes window
			RateLimitAttempts: 5,                // 5 attempts per window
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			SESFromEmail:    getEnv("AWS_SES_FROM_EMAIL", "noreply@servicepro.com"),
			S3Bucket:        getEnv("AWS_S3_BUCKET", "servicepro-uploads"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt reads an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsDuration reads an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue string) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		valueStr = defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Warning: invalid duration value for %s, using default: %s", key, defaultValue)
		defaultDuration, _ := time.ParseDuration(defaultValue)
		return defaultDuration
	}
	return value
}
