package main

import (
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
	if err != nil {
		fmt.Printf("Failed to load .env file: %s\n", err)
		return
	}

	telegramUrl := os.Getenv("TELEGRAM_BASE_URL")
	token := os.Getenv("TOKEN")
	offset := 0

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

		// make the logic for creating tenants, add users by id and the count
	}

	return offset, nil
}
