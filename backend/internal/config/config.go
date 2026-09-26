package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv          string
	PersonalAuth    bool
	Port            string
	FrontendURL     string
	DBPath          string
	TelegramToken   string
	TelegramWebhook string
	GeminiAPIKey    string
	GeminiModel     string
	JWTSecret       string
	JWTExpiryHours  int
}

func Load() *Config {
	loadEnvFile()

	expiryHours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "168"))
	personalAuth, _ := strconv.ParseBool(getEnv("PERSONAL_AUTH", "false"))

	return &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		PersonalAuth:    personalAuth,
		Port:            getEnv("PORT", "8080"),
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:5173"),
		DBPath:          getEnv("DB_PATH", "./data/pixellife.db"),
		TelegramToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramWebhook: getEnv("TELEGRAM_WEBHOOK_URL", ""),
		GeminiAPIKey:    getEnv("GEMINI_API_KEY", ""),
		GeminiModel:     getEnv("GEMINI_MODEL", "gemini-3.6-flash"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiryHours:  expiryHours,
	}
}

func loadEnvFile() {
	for _, path := range []string{".env", "../.env", "../../.env", "pixellife-tracker/.env"} {
		if err := godotenv.Load(path); err == nil {
			return
		}
	}
}

func ResolvePublicDir(baseDir string, candidates ...string) (string, error) {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		path := candidate
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("no public frontend directory found in %v", candidates)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
