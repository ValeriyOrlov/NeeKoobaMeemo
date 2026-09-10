package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DatabaseDSN   string
	JWTSecret     string
	AuthServerURL string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Load загружает переменные из .env и окружения
func Load() (*Config, error) {
	_ = godotenv.Overload()

	dbCfg := DBConfig{
		Host:     getEnv("GAME_DB_HOST", "localhost"),
		Port:     getEnv("GAME_DB_PORT", "5432"),
		User:     getEnv("GAME_DB_USER", "postgres"),
		Password: getEnv("GAME_DB_PASSWORD", "secret"),
		DBName:   getEnv("GAME_DB_NAME", "game_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	// Key-Value формат корректно экранирует спецсимволы вроде '!'
	dsn := fmt.Sprintf("host=%s port=%s user=%s password='%s' dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.SSLMode)

	// Используем DATABASE_URL только если он реально не пустой
	if envDSN := os.Getenv("DATABASE_URL"); envDSN != "" {
		dsn = envDSN
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("переменная JWT_SECRET обязательна для запуска")
	}

	port := getEnv("AUTH_PORT", "8081")

	return &Config{
		Port:          ":" + port,
		DatabaseDSN:   dsn,
		JWTSecret:     jwtSecret,
		AuthServerURL: getEnv("AUTH_SERVER_URL", "http://localhost:8080"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
