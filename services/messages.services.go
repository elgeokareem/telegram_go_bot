package services

import (
	"bot/telegram/config"
	"bot/telegram/shared"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var markdownBoldPattern = regexp.MustCompile(`\*\*([^*]+)\*\*`)

type sendMessageRequest struct {
	ChatID      int64                `json:"chat_id"`
	Text        string               `json:"text"`
	ReplyMarkup inlineKeyboardMarkup `json:"reply_markup"`
}

type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

type inlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type botCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type setMyCommandsRequest struct {
	Commands []botCommand `json:"commands"`
}

func RegisterBotCommands() error {
	env := config.Current
	baseUrl := env.TelegramBaseURL + env.Token + "/setMyCommands"
	payload := setMyCommandsRequest{
		Commands: []botCommand{
			{Command: "command", Description: "Show command help"},
			{Command: "ask_catholic_church", Description: "Ask a Catholic teaching question"},
			{Command: "new_event", Description: "Open the event form"},
			{Command: "set_birthday", Description: "Reply with DD-MM-YYYY to save a birthday"},
			{Command: "show_events", Description: "Show all active events in this group"},
			{Command: "delete_event", Description: "Delete an event by ID (admins only)"},
			{Command: "lovedusers", Description: "Show users with the most positive karma"},
			{Command: "hatedusers", Description: "Show users with the most negative karma"},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := shared.CustomClient.Post(baseUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned status %d for setMyCommands: %s", resp.StatusCode, string(body))
	}

	return nil
}

func SendMessage(chatId int64, message string) error {
	// Define the base URL
	env := config.Current
	token := env.Token
	telegramUrl := env.TelegramBaseURL
	baseUrl := telegramUrl + token + "/sendMessage"

	// Create the data for the API request
	data := url.Values{}
	data.Add("chat_id", strconv.FormatInt(chatId, 10))
	data.Add("text", message)

	// Append the data to the URL
	completeUrl := baseUrl + "?" + data.Encode()

	// Send the HTTP request with retry logic
	resp, err := shared.CustomClient.Get(completeUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d for sendMessage", resp.StatusCode)
	}

	return nil
}

func SendMessageWithReply[T ~int | ~int64](chatId int64, replyToMessageId T, message string) error {
	// Define the base URL
	env := config.Current
	token := env.Token
	telegramUrl := env.TelegramBaseURL
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d for sendMessage with reply", resp.StatusCode)
	}

	return nil
}

func SendMessageWithReplyParseMode[T ~int | ~int64](chatId int64, replyToMessageId T, message string, parseMode string) error {
	env := config.Current
	token := env.Token
	telegramUrl := env.TelegramBaseURL
	baseUrl := telegramUrl + token + "/sendMessage"

	data := url.Values{}
	data.Add("chat_id", strconv.FormatInt(chatId, 10))
	data.Add("text", message)
	data.Add("reply_to_message_id", strconv.FormatInt(int64(replyToMessageId), 10))
	data.Add("parse_mode", parseMode)
	data.Add("disable_web_page_preview", "true")

	completeUrl := baseUrl + "?" + data.Encode()
	resp, err := shared.CustomClient.Get(completeUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d for sendMessage with parse mode", resp.StatusCode)
	}

	return nil
}

func SendLongMessageWithReply[T ~int | ~int64](chatId int64, replyToMessageId T, message string) error {
	const telegramMessageLimit = 4096
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil
	}

	for len(trimmedMessage) > 0 {
		chunk := trimmedMessage
		if len(chunk) > telegramMessageLimit {
			chunk = trimmedMessage[:telegramMessageLimit]
			if lastNewline := strings.LastIndex(chunk, "\n"); lastNewline > 0 {
				chunk = chunk[:lastNewline]
			}
		}

		if err := SendMessageWithReply(chatId, replyToMessageId, chunk); err != nil {
			return err
		}

		trimmedMessage = strings.TrimSpace(strings.TrimPrefix(trimmedMessage, chunk))
	}

	return nil
}

func SendLongHTMLMessageWithReply[T ~int | ~int64](chatId int64, replyToMessageId T, message string) error {
	const telegramMessageLimit = 3500
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil
	}

	for len(trimmedMessage) > 0 {
		chunk := trimmedMessage
		if len(chunk) > telegramMessageLimit {
			chunk = trimmedMessage[:telegramMessageLimit]
			if lastNewline := strings.LastIndex(chunk, "\n"); lastNewline > 0 {
				chunk = chunk[:lastNewline]
			}
		}

		if err := SendMessageWithReplyParseMode(chatId, replyToMessageId, telegramHTMLFromMarkdown(chunk), "HTML"); err != nil {
			return err
		}

		trimmedMessage = strings.TrimSpace(strings.TrimPrefix(trimmedMessage, chunk))
	}

	return nil
}

func telegramHTMLFromMarkdown(message string) string {
	escaped := html.EscapeString(message)
	return markdownBoldPattern.ReplaceAllString(escaped, "<b>$1</b>")
}

func BuildEventsWebAppURL(chatId int64, userId int64) string {
	env := config.Current
	parsedURL, err := url.Parse(env.TelegramWebAppURL)
	if err != nil {
		return env.TelegramWebAppURL
	}

	query := parsedURL.Query()
	query.Set("ctx", createSignedWebAppContext(chatId, userId))
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

func SendEventsWebAppMessage(chatId int64, userId int64) error {
	env := config.Current
	baseUrl := env.TelegramBaseURL + env.Token + "/sendMessage"
	webAppURL := BuildEventsWebAppURL(chatId, userId)
	payload := sendMessageRequest{
		ChatID: chatId,
		Text:   "Create a new event from the Telegram Web App.",
		ReplyMarkup: inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{{{
			Text: "Create event",
			URL:  webAppURL,
		}}}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := shared.CustomClient.Post(baseUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned status %d for sendMessage with web app: %s", resp.StatusCode, string(body))
	}

	return nil
}

func createSignedWebAppContext(chatId int64, userId int64) string {
	expiresAt := time.Now().Add(15 * time.Minute).Unix()
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d:%d:%d", chatId, userId, expiresAt)))

	mac := hmac.New(sha256.New, []byte(config.Current.WebAppContextSecret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return payload + "." + signature
}
