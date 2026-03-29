package main

import (
	"bot/telegram/config"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run migrate.go <up|down>")
	}

	if err := config.Init(); err != nil {
		log.Fatalf("Failed to load environment configuration: %v", err)
	}

	env := config.Current

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env.DBUser,
		env.DBPassword,
		env.DBHost,
		env.DBPort,
		env.DBName)

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
