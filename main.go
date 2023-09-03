package main

import (
	"bot/telegram/services"
	"fmt"
	"log"
	"os"
	"runtime"

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
	_, errDb := services.CreateDbConnection(dbName)
	if errDb != nil {
		fmt.Printf("Error connecting to the db. %s", errDb)
		return
	}

	iteration := 0

	for {
		iteration++
		fmt.Println("OFFSET COMENZANDO", iteration)
		fmt.Println("Number of goroutines:", runtime.NumGoroutine())
		offset, err = services.ProcessTelegramMessages(telegramUrl, token, offset)
		fmt.Println(offset)
		if err != nil {
			// log.Fatal(err). log.Fatal terminates the program
			log.Println("ERROR: ", err)
		}
	}
}
