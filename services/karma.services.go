package services

import (
	"bot/telegram/shared"
	"bot/telegram/structs"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func AddKarmaToUser(update structs.Update, karmaValue *int, conn *pgx.Conn) error {
	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID
	messageToGiveKarma := update.Message.ReplyToMessage.From

	totalKarma, err := UpsertUserKarma(
		conn,
		messageToGiveKarma.ID,
		chatId,
		messageToGiveKarma.FirstName,
		messageToGiveKarma.LastName,
		messageToGiveKarma.Username,
		*karmaValue, // Dereference karmaValue here
		0,           // karmaGivenIncrement for receiver
		0,           // karmaTakenIncrement for receiver
	)
	if err != nil {
		return err
	}

	// Update karma_given or karma_taken for the sender
	senderID := update.Message.From.ID
	senderGroupID := update.Message.Chat.ID
	senderFirstName := update.Message.From.FirstName
	senderLastName := update.Message.From.LastName
	senderUsername := update.Message.From.Username

	senderKarmaGivenIncrement := 0
	senderKarmaTakenIncrement := 0

	if *karmaValue > 0 {
		senderKarmaGivenIncrement = 1
	} else if *karmaValue < 0 {
		senderKarmaTakenIncrement = 1
	}

	_, err = UpsertUserKarma(
		conn,
		senderID,
		senderGroupID,
		senderFirstName,
		senderLastName,
		senderUsername,
		0, // karmaValue for sender (not changing sender's main karma score)
		senderKarmaGivenIncrement,
		senderKarmaTakenIncrement,
	)
	if err != nil {
		CreateErrorRecord(conn, ErrorRecordInput{
			GroupID:    senderGroupID,
			SenderID:   senderID,
			ReceiverID: messageToGiveKarma.ID,
			Error:      fmt.Sprintf("error updating karma_given/taken for sender: %v", err),
		})
		return fmt.Errorf("error updating karma_given/taken for sender: %w", err)
	}

	successMessage := fmt.Sprintf("Karma added to %s. Total karma: %d", messageToGiveKarma.FirstName, totalKarma)
	if err := SendMessageWithReply(chatId, replyToMessageId, successMessage); err != nil {
		CreateErrorRecord(conn, ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   update.Message.From.ID,
			ReceiverID: messageToGiveKarma.ID,
			Error:      err.Error(),
		})
	}

	// TODO: Add karma restrictrions per group

	return nil
}

// makeTelegramAPIRequest performs HTTP request to Telegram API with retry logic
func makeTelegramAPIRequest(url string) (*http.Response, error) {
	var lastErr error

	// Retry up to 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		response, err := shared.CustomClient.Get(url)
		if err != nil {
			lastErr = err
			if attempt < 2 { // Don't sleep on the last attempt
				waitTime := time.Duration(attempt+1) * time.Second
				fmt.Printf("Telegram API request failed (attempt %d/3): %s. Retrying in %v...\n", attempt+1, err, waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		// Check if the response status is OK
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			lastErr = fmt.Errorf("telegram API returned status %d", response.StatusCode)
			if attempt < 2 {
				waitTime := time.Duration(attempt+1) * time.Second
				fmt.Printf("Telegram API returned error status %d (attempt %d/3). Retrying in %v...\n", response.StatusCode, attempt+1, waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		return response, nil
	}

	return nil, fmt.Errorf("telegram API request failed after 3 attempts: %w", lastErr)
}

func ProcessTelegramMessages(telegramUrl string, token string, offset int, conn *pgx.Conn) (int, error) {
	longPollTimeout := 25
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=%d", telegramUrl, token, offset, longPollTimeout)

	response, err := makeTelegramAPIRequest(completeUrl)
	if err != nil {
		return offset, fmt.Errorf("failed to get updates from Telegram API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return offset, fmt.Errorf("failed to read response body: %w", err)
	}

	var result structs.UpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return offset, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	newOffset := offset
	for _, update := range result.Result {
		newOffset = update.UpdateID + 1

		if update.Message == nil || update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From == nil {
			continue // Skip this update if it's not a reply or doesn't have a sender in the reply
		}

		chatId := update.Message.Chat.ID
		senderMessageId := update.Message.MessageID

		// Handle /lovedusers command
		if strings.HasPrefix(update.Message.Text, "/lovedusers@WillibertoBot") {
			lovedUsers, err := GetMostLovedUsers(conn)
			if err != nil {
				CreateErrorRecord(conn, ErrorRecordInput{Error: err.Error()})
				continue // Skip this update instead of returning
			}
			fmt.Println(lovedUsers)
			continue // Move to next update
		}

		// Handle /hatedusers command
		if strings.HasPrefix(update.Message.Text, "/hatedusers@WillibertoBot") {
			hatedUsers, err := GetMostHatedUsers(conn)
			if err != nil {
				CreateErrorRecord(conn, ErrorRecordInput{Error: err.Error()})
				continue // Skip this update instead of returning
			}
			fmt.Println(hatedUsers)
			continue // Move to next update
		}

		isPlusMinusOne, karmaValue := shared.ParsePlusMinusOneFromMessage(update.Message.Text)
		if !isPlusMinusOne {
			continue
		}

		// Validations for giving karma
		if err := KarmaValidations(update, conn); err != nil {
			CreateErrorRecord(conn, ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   update.Message.From.ID,
				ReceiverID: update.Message.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
			continue // Skip this update instead of returning
		}

		if err := AddKarmaToUser(update, karmaValue, conn); err != nil {
			errorInput := ErrorRecordInput{
				SenderID:   update.Message.ReplyToMessage.From.ID,
				ReceiverID: update.Message.From.ID,
				GroupID:    chatId,
				Error:      err.Error(),
			}
			if err := SendMessageWithReply(chatId, senderMessageId, "Error adding karma"); err != nil {
				CreateErrorRecord(conn, ErrorRecordInput{
					GroupID:    chatId,
					SenderID:   update.Message.From.ID,
					ReceiverID: update.Message.ReplyToMessage.From.ID,
					Error:      err.Error(),
				})
				continue
			}
			CreateErrorRecord(conn, errorInput)
			continue // Skip this update instead of returning
		}
	}

	return newOffset, nil
}
