package main

import (
	"fmt"
	"strings"
	"time"

	"bot/telegram/config"
	"bot/telegram/services"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	telegramUrl := config.Env.TelegramBaseURL
	dbName := config.Env.DBName
	token := config.Env.Token

	if telegramUrl == "" || dbName == "" || token == "" {
		fmt.Println("Missing required environment variables: TELEGRAM_BASE_URL, DB_NAME, or TOKEN")
		return
	}

	offset := 0

	// Main processing loop with connection recovery
	for {
		// Get a connection from the pool
		conn, err := services.GlobalPoolManager.GetConnectionFromPool(dbName)
		if err != nil {
			fmt.Printf("Failed to get database connection: %s. Retrying in 30 seconds...\n", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Process Telegram messages
		newOffset, err := services.ProcessTelegramMessages(telegramUrl, token, offset, conn.Conn())
		conn.Release()

		if err != nil {
			fmt.Printf("Error processing Telegram messages: %s\n", err)

			// Log the error to database if possible
			errorInput := services.ErrorRecordInput{
				Error: err.Error(),
			}
			if dbErr := services.CreateErrorRecord(conn.Conn(), errorInput); dbErr != nil {
				fmt.Printf("Failed to log error to database: %s\n", dbErr)
			}

			// If it's a network-related error, wait longer before retrying
			if isNetworkError(err) {
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

// isNetworkError checks if the error is network-related
func isNetworkError(err error) bool {
	errStr := err.Error()
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"network is unreachable",
		"no such host",
		"failed to get updates",
		"Telegram API returned status",
	}

	for _, networkErr := range networkErrors {
		if strings.Contains(strings.ToLower(errStr), networkErr) {
			return true
		}
	}
	return false
}
