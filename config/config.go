package config

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type EnvVar struct {
	TelegramBaseURL string `env:"TELEGRAM_BASE_URL"`
	Token           string `env:"TOKEN"`
	DBUser          string `env:"DB_USER"`
	DBPassword      string `env:"DB_PASSWORD"`
	DBHost          string `env:"DB_HOST"`
	DBPort          string `env:"DB_PORT"`
	DBName          string `env:"DB_NAME"`
	DBSchema        string `env:"DB_SCHEMA"`
}

var Env EnvVar

func getEnvPath() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "..", ".env")
}

func LoadEnv() {
	if err := godotenv.Load(getEnvPath()); err != nil {
		log.Println("No .env file found")
	}

	if err := env.Parse(&Env); err != nil {
		log.Fatalf("Failed to parse env: %v", err)
	}
}
