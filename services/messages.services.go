package services

import (
	"bot/telegram/shared"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func SendMessage(chatId int64, message string) error {
	// Define the base URL
	token := os.Getenv("TOKEN")
	telegramUrl := os.Getenv("TELEGRAM_BASE_URL")
	baseUrl := telegramUrl + token + "/sendMessage"

	// Create the data for the API request
	data := url.Values{}
	data.Add("chat_id", strconv.FormatInt(chatId, 10))
	data.Add("text", message)

	// Append the data to the URL
	completeUrl := baseUrl + "?" + data.Encode()

	// Send the HTTP request
	resp, err := http.Get(completeUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func SendMessageWithReply[T ~int | ~int64](chatId int64, replyToMessageId T, message string) error {
	// Define the base URL
	token := os.Getenv("TOKEN")
	telegramUrl := os.Getenv("TELEGRAM_BASE_URL")
	baseUrl := telegramUrl + token + "/sendMessage"

	// Create the data for the API request
	data := url.Values{}
	data.Add("chat_id", strconv.FormatInt(chatId, 10))
	data.Add("text", message)
	data.Add("reply_to_message_id", strconv.FormatInt(int64(replyToMessageId), 10))

	// Append the data to the URL
	completeUrl := baseUrl + "?" + data.Encode()

	// Send the HTTP request
	resp, err := shared.CustomClient.Get(completeUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
