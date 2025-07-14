# Makefile for Bot Telegram

# Load environment variables from .env file
-include .env

# Ensure that the required variables are exported to be available for the shell commands
export DB_USER DB_PASSWORD DB_HOST DB_PORT DB_NAME

# Construct the database URL from environment variables
DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: migrate-up migrate-down migrate-create build

## Run all pending up migrations
migrate-up:
	@echo "Running up migrations..."
	@go run migrate.go up

## Revert the last applied migration
migrate-down:
	@echo "Running down migrations..."
	@go run ./database/migrate.go down

build:
	@go build -o telegram_go_bot .

## Create a new migration file. Requires a 'name' argument.
## Example: make migrate-create name=add_user_table
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	@echo "Creating migration: $(name)..."
	@migrate create -ext sql -dir database/migrations $(name)

