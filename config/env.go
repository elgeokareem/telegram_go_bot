package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Env struct {
	TelegramBaseURL     string
	Token               string
	TelegramWebAppURL   string
	WebAppContextSecret string
	DBSchema            string
	DBName              string
	DBUser              string
	DBPassword          string
	DBHost              string
	DBPort              string
	DBDefaultName       string
}

var Current Env

func Init() error {
	env, err := Load()
	if err != nil {
		return err
	}

	Current = env
	return nil
}

func Load() (Env, error) {
	_ = godotenv.Load(".env")

	env := Env{
		TelegramBaseURL:     getEnvOrDefault("TELEGRAM_BASE_URL", "https://api.telegram.org/bot"),
		Token:               strings.TrimSpace(os.Getenv("TOKEN")),
		TelegramWebAppURL:   getEnvOrDefault("TELEGRAM_WEB_APP_URL", "https://telegram.william-vegas.com/events-new"),
		WebAppContextSecret: strings.TrimSpace(os.Getenv("WEB_APP_CONTEXT_SECRET")),
		DBSchema:            getEnvOrDefault("DB_SCHEMA", "postgres"),
		DBName:              strings.TrimSpace(os.Getenv("DB_NAME")),
		DBUser:              getEnvOrDefault("DB_USER", "postgres"),
		DBPassword:          strings.TrimSpace(os.Getenv("DB_PASSWORD")),
		DBHost:              getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:              getEnvOrDefault("DB_PORT", "5432"),
		DBDefaultName:       getEnvOrDefault("DB_DEFAULT_NAME", "postgres"),
	}

	if env.DBName == "" {
		return env, fmt.Errorf("missing required environment variable: DB_NAME")
	}

	return env, nil
}

func (e Env) ValidateBot() error {
	missing := make([]string, 0, 2)

	if e.Token == "" {
		missing = append(missing, "TOKEN")
	}

	if e.WebAppContextSecret == "" {
		missing = append(missing, "WEB_APP_CONTEXT_SECRET")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
