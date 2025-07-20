package services

import (
	"bot/telegram/structs"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func KarmaValidations(update structs.Update, conn *pgx.Conn) error {
	if update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From == nil {
		return errors.New("no reply or sender")
	}

	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID

	// If user try to give karma to itself
	if update.Message.ReplyToMessage.From.ID == update.Message.From.ID {
		err := SendMessageWithReply(chatId, replyToMessageId, "Wew. You can't give karma to yourself dummy ~")
		if err != nil {
			CreateErrorRecord(conn, ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   update.Message.From.ID,
				ReceiverID: update.Message.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
		}
		return errors.New("can't give karma to yourself")
	}

	// If user is not inside the time frame
	if err := UpdateKarmaGivenTimeOfUser(conn, update.Message); err != nil {
		return err
	}

	return nil
}

func UpdateKarmaGivenTimeOfUser(conn *pgx.Conn, currentMessage *structs.Message) error {
	chatId := currentMessage.Chat.ID
	replyToMessageId := currentMessage.ReplyToMessage.MessageID

	fmt.Printf("Updating karma given time for user %d in group %d\n", currentMessage.From.ID, currentMessage.Chat.ID)

	var lastMessageDateTime time.Time
	err := conn.QueryRow(context.Background(), "SELECT last_karma_given FROM users_ranking WHERE user_id = $1 AND group_id = $2", currentMessage.From.ID, currentMessage.Chat.ID).Scan(&lastMessageDateTime)
	fmt.Printf("Error after queryRow/scan: %v\n", err)
	fmt.Printf("1 Last message date time: %s\n", lastMessageDateTime)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("error querying last_karma_given: %w", err)
	}

	fmt.Printf("2 Last message date time: %s\n", lastMessageDateTime)

	if err == pgx.ErrNoRows {
		_, errUpsert := UpsertUserKarma(
			conn,
			currentMessage.From.ID,
			chatId,
			currentMessage.From.FirstName,
			currentMessage.From.LastName,
			currentMessage.From.Username,
			0,
		)

		if errUpsert != nil {
			return err
		}

		return nil
	}

	thresholdMessageLimit := 60 * time.Second
	if time.Since(lastMessageDateTime) < thresholdMessageLimit {
		err := SendMessageWithReply(chatId, replyToMessageId, "Whoops you are not allowed to give karma yet :(")
		if err != nil {
			CreateErrorRecord(conn, ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   currentMessage.From.ID,
				ReceiverID: currentMessage.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
		}
		return errors.New("can't give karma yet")
	}

	return nil
}
