package main

import (
	"bot/telegram/services"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// TODO: Change the chatId in the DB to be string and not number. This because some ids can be negative and have problems with SQL

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
	_, errDb := services.CreateDbConnection(dbName)
	if errDb != nil {
		fmt.Printf("Error connecting to the db. %s", errDb)
		return
	}

	for {
		offset, err = services.GetKarmaUpdates(telegramUrl, token, offset)
		fmt.Println(offset)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second * 3)
	}
}
