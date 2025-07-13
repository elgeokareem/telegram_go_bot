package services

import (
	"bot/telegram/shared"
	"bot/telegram/structs"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func AddKarmaToUser(update structs.Update, karmaValue *int, conn *pgx.Conn) error {
	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID
	messageToGiveKarma := update.Message.ReplyToMessage.From

	// TODO: Probably put it in another place. All validations should be together.
	// Check if User can give karma given time constriction
	// CheckLastTimeUserGaveKarma(conn, currentMessage)

	sqlToAddKarma := `
		INSERT INTO users_ranking (user_id, group_id, first_name, last_name, username, karma, last_karma_given)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			username = EXCLUDED.username,
			karma = users_ranking.karma + $6,
			last_karma_given = EXCLUDED.last_karma_given
    RETURNING karma
	`

	// currentMessageJSON, _ := json.MarshalIndent(update.Message, "", "  ")
	// fmt.Printf("CURRENT MESSAGE: %s\n", string(currentMessageJSON))
	//
	// messageToGiveKarmaJSON, _ := json.MarshalIndent(messageToGiveKarma, "", "  ")
	// fmt.Printf("MESSAGE TO GIVE KARMA: %s\n", string(messageToGiveKarmaJSON))

	var totalKarma int
	err := conn.QueryRow(
		context.Background(),
		sqlToAddKarma,
		messageToGiveKarma.ID,
		chatId,
		messageToGiveKarma.FirstName,
		messageToGiveKarma.LastName,
		messageToGiveKarma.Username,
		karmaValue,
		time.Now(),
	).Scan(&totalKarma)
	if err != nil {
		return err
	}

	successMessage := fmt.Sprintf("Karma added to %s. Total karma: %d", messageToGiveKarma.FirstName, totalKarma)
	SendMessageWithReply(chatId, replyToMessageId, successMessage)

	return nil
}

// Check when was the last time the user gave karma.
func CheckLastTimeUserGaveKarma(conn *pgx.Conn, currentMessage *structs.Message) error {
	// TODO: For the future add a flag in the table with true or false something like.
	sqlToAddKarmaGiver := `
		SELECT last_karma_given FROM users_ranking ur WHERE ur.user_id = $1
	`

	_, err := conn.Exec(
		context.Background(),
		sqlToAddKarmaGiver,
		currentMessage.From.ID,
	)
	if err != nil {
		return fmt.Errorf("error => CheckIfUserCanGiveKarma:  %w", err)
	}

	return nil
}

func ProcessTelegramMessages(telegramUrl string, token string, offset int, conn *pgx.Conn) (int, error) {
	longPollTimeout := 25
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=%d", telegramUrl, token, offset, longPollTimeout)

	fmt.Println("Fetching updates from:", completeUrl)

	response, err := shared.CustomClient.Get(completeUrl)
	if err != nil {
		fmt.Println("ERROR http.Get(completeUrl)", err)
		return offset, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return offset, err
	}

	var result structs.UpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return offset, err
	}

	newOffset := offset
	for _, update := range result.Result {
		newOffset = update.UpdateID + 1

		if update.Message == nil {
			continue // Skip this update instead of returning
		}

		chatId := update.Message.Chat.ID
		senderMessageId := update.Message.MessageID

		// Handle /lovedusers command
		if strings.HasPrefix(update.Message.Text, "/lovedusers@WillibertoBot") {
			lovedUsers, err := GetMostLovedUsers(conn)
			if err != nil {
				fmt.Println("ERROR loved users")
				continue // Skip this update instead of returning
			}
			fmt.Println(lovedUsers)
			continue // Move to next update
		}

		// Handle /hatedusers command
		if strings.HasPrefix(update.Message.Text, "/hatedusers@WillibertoBot") {
			hatedUsers, err := GetMostHatedUsers(conn)
			if err != nil {
				fmt.Println("ERROR loved users")
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
			continue // Skip this update instead of returning
		}

		if err := AddKarmaToUser(update, karmaValue, conn); err != nil {
			errorInput := ErrorRecordInput{
				SenderID:   update.Message.ReplyToMessage.From.ID,
				ReceiverID: update.Message.From.ID,
				GroupID:    chatId,
				Error:      err.Error(),
			}
			SendMessageWithReply(chatId, senderMessageId, "Error adding karma")
			CreateErrorRecord(conn, errorInput)
			continue // Skip this update instead of returning
		}
	}

	return newOffset, nil
}
