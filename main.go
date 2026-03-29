package main

import (
	"bot/telegram/config"
	"bot/telegram/errors"
	"bot/telegram/services"
	"fmt"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if err := config.Init(); err != nil {
		fmt.Printf("Failed to load environment configuration: %s\n", err)
		return
	}

	env := config.Current

	if err := env.ValidateBot(); err != nil {
		fmt.Println(err.Error())
		return
	}

	offset := 0

	// Main processing loop with connection recovery
	for {
		// Get a connection from the pool
		conn, err := services.GlobalPoolManager.GetConnectionFromPool(env.DBName)
		if err != nil {
			fmt.Printf("Failed to get database connection: %s. Retrying in 30 seconds...\n", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Process Telegram messages
		pgConn := conn.Conn()
		newOffset, err := services.ProcessTelegramMessages(env.TelegramBaseURL, env.Token, offset, pgConn)

		if err != nil {
			fmt.Printf("Error processing Telegram messages: %s\n", err)

			// Log the error to database if possible
			errorInput := errors.ErrorRecordInput{
				Error: err.Error(),
			}
			if dbErr := errors.CreateErrorRecord(pgConn, errorInput); dbErr != nil {
				fmt.Printf("Failed to log error to database: %s\n", dbErr)
			}
		}

		conn.Release()

		if err != nil {

			// If it's a network-related error, wait longer before retrying
			if errors.IsNetworkError(err) {
				fmt.Println("Network error detected, waiting 10 seconds before retry...")
				time.Sleep(10 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
			}
		} else {
			offset = newOffset
			time.Sleep(1 * time.Second)
		}
	}
}
