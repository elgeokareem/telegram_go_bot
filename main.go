package main

import (
	"bot/telegram/services"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
		errorInput := services.ErrorRecordInput{
			Error: errDb.Error(),
		}
		services.CreateErrorRecord(conn, errorInput)
		return
	}

	for {
		offset, err = services.ProcessTelegramMessages(telegramUrl, token, offset, conn)
		if err != nil {
			// log.Fatal(err). log.Fatal terminates the program
			log.Println("ERROR: ", err)
			errorInput := services.ErrorRecordInput{
				Error: err.Error(),
			}
			services.CreateErrorRecord(conn, errorInput)
		}
		time.Sleep(1 * time.Second)
	}
}
