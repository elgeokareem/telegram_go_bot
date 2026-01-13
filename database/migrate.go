package main

import (
	"fmt"
	"log"
	"os"

	"bot/telegram/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run migrate.go <up|down>")
	}

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.Env.DBUser,
		config.Env.DBPassword,
		config.Env.DBHost,
		config.Env.DBPort,
		config.Env.DBName)

	m, err := migrate.New(
		"file://database/migrations",
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	action := os.Args[1]
	switch action {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run up migrations: %v", err)
		}
		log.Println("Up migrations applied successfully!")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run down migrations: %v", err)
		}
		log.Println("Down migrations applied successfully!")
	default:
		log.Fatalf("Invalid action: %s. Use 'up' or 'down'", action)
	}
}
