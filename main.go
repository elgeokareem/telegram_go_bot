package main

import (
	"bot/telegram/services"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	telegramUrl := os.Getenv("TELEGRAM_BASE_URL")
	dbName := os.Getenv("DB_NAME")
	token := os.Getenv("TOKEN")
	offset := 0

	if err != nil {
		fmt.Printf("Failed to load .env file: %s\n", err)
		return
	}

	// Init main DB
	conn, errDb := services.CreateDbConnection(dbName)
	if errDb != nil {
		fmt.Printf("Error connecting to the db. %s", errDb)
		return
	}

	iteration := 0

	for {
		iteration++
		offset, err = services.ProcessTelegramMessages(telegramUrl, token, offset, conn)
		if err != nil {
			// log.Fatal(err). log.Fatal terminates the program
			log.Println("ERROR: ", err)
		}
		time.Sleep(1 * time.Second)
	}
}
