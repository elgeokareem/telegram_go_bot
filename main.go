package main

import (
	"bot/telegram/services"
	"bot/telegram/structs"
	"bot/telegram/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	_, errkek := services.CreateDbConnection(dbName)
	if err != nil {
		fmt.Printf("Error connecting to the db. %s", errkek)
		return
	}

	for {
		offset, err = getUpdates(telegramUrl, token, offset)
		fmt.Println(offset)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second * 3)
	}
}

func getUpdates(telegramUrl string, token string, offset int) (int, error) {
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d", telegramUrl, token, offset)

	response, err := http.Get(completeUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result structs.UpdateResponse
	errBody := json.Unmarshal(body, &result)
	if errBody != nil {
		return offset, err
	}

	for _, update := range result.Result {
		updateID := update.UpdateID
		if updateID >= offset {
			offset = updateID + 1
		}

		isPlusOne := utils.ParsePlusOneFromMessage(update.Message.Text)

		if !isPlusOne {
			continue
		}

		// make the logic for creating tenants, add users by id and the count
		// we can parse
	}

	return offset, nil
}
