package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	RateLimit  RateLimitConfig
	CORS       CORSConfig
	Email      EmailConfig
	Upload     UploadConfig
	Pagination PaginationConfig
	S3         S3Config
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	DSN      string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Addr     string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type RateLimitConfig struct {
	RPM             int
	AuthRPM         int
	LockoutAttempts int
	LockoutDuration time.Duration
}

type CORSConfig struct {
	Origins []string
}

type EmailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type UploadConfig struct {
	MaxSize      int64
	AllowedTypes []string
}

type PaginationConfig struct {
	DefaultPageSize int
	MaxPageSize     int
}

type S3Config struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

// Load loads configuration from environment
func Load() (*Config, error) {
	_ = godotenv.Load()

	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	lockoutDuration, _ := time.ParseDuration(getEnv("ACCOUNT_LOCKOUT_DURATION", "15m"))

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8089"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", defaultDBUser()),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "auto_store"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "change-me-in-production"),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		RateLimit: RateLimitConfig{
			RPM:             getEnvInt("RATE_LIMIT_RPM", 60),
			AuthRPM:         getEnvInt("AUTH_RATE_LIMIT_RPM", 10),
			LockoutAttempts: getEnvInt("ACCOUNT_LOCKOUT_ATTEMPTS", 5),
			LockoutDuration: lockoutDuration,
		},
		CORS: CORSConfig{
			Origins: getEnvSlice("CORS_ORIGINS", []string{"*"}),
		},
		Email: EmailConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnvInt("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@auto-store.com"),
		},
		Upload: UploadConfig{
			MaxSize:      getEnvInt64("UPLOAD_MAX_SIZE", 5242880),
			AllowedTypes: getEnvSlice("UPLOAD_ALLOWED_TYPES", []string{"image/jpeg", "image/png", "image/webp"}),
		},
		Pagination: PaginationConfig{
			DefaultPageSize: getEnvInt("DEFAULT_PAGE_SIZE", 20),
			MaxPageSize:     getEnvInt("MAX_PAGE_SIZE", 100),
		},
		S3: S3Config{
			Bucket:    getEnv("S3_BUCKET", ""),
			Region:    getEnv("S3_REGION", ""),
			AccessKey: getEnv("S3_ACCESS_KEY", ""),
			SecretKey: getEnv("S3_SECRET_KEY", ""),
			Endpoint:  getEnv("S3_ENDPOINT", ""),
		},
	}

	cfg.Database.DSN = "host=" + cfg.Database.Host + " port=" + cfg.Database.Port +
		" user=" + cfg.Database.User + " password=" + cfg.Database.Password +
		" dbname=" + cfg.Database.DBName + " sslmode=" + cfg.Database.SSLMode
	cfg.Redis.Addr = cfg.Redis.Host + ":" + cfg.Redis.Port

	return cfg, nil
}

// defaultDBUser returns the OS username on macOS/Unix (e.g. "Apple") so local Postgres works without .env; "postgres" in Docker.
func defaultDBUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "postgres"
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvSlice(key string, defaultVal []string) []string {
	if v := os.Getenv(key); v != "" {
		return strings.Split(strings.TrimSpace(v), ",")
	}
	return defaultVal
}
