package services

import (
	"bot/telegram/errors"
	"bot/telegram/structs"
	"context"
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func KarmaValidations(update structs.Update, conn *pgx.Conn) error {
	if update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From == nil {
		return stdErrors.New("no reply or sender")
	}

	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID

	// If user try to give karma to itself
	if update.Message.ReplyToMessage.From.ID == update.Message.From.ID {
		err := SendMessageWithReply(chatId, replyToMessageId, "Wew. You can't give karma to yourself dummy ~")
		if err != nil {
			errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   update.Message.From.ID,
				ReceiverID: update.Message.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
		}
		return stdErrors.New("can't give karma to yourself")
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

	if err != nil && err != pgx.ErrNoRows {
		errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   currentMessage.From.ID,
			ReceiverID: currentMessage.ReplyToMessage.From.ID,
			Error:      err.Error(),
		})
		return fmt.Errorf("error querying last_karma_given: %w", err)
	}

	fmt.Printf("Retrieved lastMessageDateTime: %v\n", lastMessageDateTime)

	if err == pgx.ErrNoRows {
		// If no record found, it means this is the first time this user is giving karma.
		// Allow it, and the last_karma_given will be set when karma is successfully added.
		return nil
	}

	thresholdMessageLimit := 60 * time.Second
	fmt.Printf("Time since last message: %v\n", time.Since(lastMessageDateTime))
	if time.Since(lastMessageDateTime) < thresholdMessageLimit {
		err := SendMessageWithReply(chatId, replyToMessageId, "Whoops you are not allowed to give karma yet :(")
		if err != nil {
			errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   currentMessage.From.ID,
				ReceiverID: currentMessage.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
		}
		return stdErrors.New("can't give karma yet")
	}

	// Update last_karma_given for the sender
	fmt.Printf("Executing UPDATE for last_karma_given for sender %d in group %d\n", currentMessage.From.ID, currentMessage.Chat.ID)
	_, err = conn.Exec(context.Background(), "UPDATE users_ranking SET last_karma_given = $3 WHERE user_id = $1 AND group_id = $2", currentMessage.From.ID, currentMessage.Chat.ID, time.Now().UTC())
	if err != nil {
		errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   currentMessage.From.ID,
			ReceiverID: currentMessage.ReplyToMessage.From.ID,
			Error:      err.Error(),
		})
		return fmt.Errorf("error updating last_karma_given for sender: %w", err)
	}
	fmt.Printf("Updated last_karma_given for sender %d in group %d\n", currentMessage.From.ID, currentMessage.Chat.ID)

	return nil
}
