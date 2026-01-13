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
			_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   update.Message.From.ID,
				ReceiverID: update.Message.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
		}
		return stdErrors.New("can't give karma to yourself")
	}

	// If user is not inside the time frame
	if err := DbUserRestrictions(conn, update.Message); err != nil {
		return err
	}

	return nil
}

func DbUserRestrictions(conn *pgx.Conn, currentMessage *structs.Message) error {
	chatId := currentMessage.Chat.ID
	replyToMessageId := currentMessage.ReplyToMessage.MessageID
	receiverId := currentMessage.ReplyToMessage.From.ID

	var lastMessageDateTime time.Time
	var allowedToGiveKarma bool

	validationSenderSql := "SELECT last_karma_given, allowed_to_give_karma FROM users_ranking WHERE user_id = $1 AND group_id = $2"
	err := conn.QueryRow(context.Background(), validationSenderSql, currentMessage.From.ID, currentMessage.Chat.ID).Scan(&lastMessageDateTime, &allowedToGiveKarma)

	if err != nil && err != pgx.ErrNoRows {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   currentMessage.From.ID,
			ReceiverID: currentMessage.ReplyToMessage.From.ID,
			Error:      err.Error(),
		})
		return fmt.Errorf("error querying last_karma_given: %w", err)
	}

	if err == pgx.ErrNoRows {
		// If no record found, it means this is the first time this user is giving karma.
		// Allow it, and the last_karma_given will be set when karma is successfully added.
		return nil
	}

	if !allowedToGiveKarma {
		_ = SendMessageWithReply(chatId, replyToMessageId, "Sorry bro you can't give aura points around here.")
		return stdErrors.New("can't give karma to yourself")
	}

	var allowedToReceiveKarma bool
	validationReceiverSql := "SELECT allowed_to_receive_karma FROM users_ranking WHERE user_id = $1 AND group_id = $2"
	err = conn.QueryRow(context.Background(), validationReceiverSql, receiverId, currentMessage.Chat.ID).Scan(&allowedToReceiveKarma)

	if !allowedToReceiveKarma {
		_ = SendMessageWithReply(chatId, replyToMessageId, "Sorry bro this person can't receive aura points.")
		return stdErrors.New("receiver not allowed to receive karma")
	}

	if err != nil && err != pgx.ErrNoRows {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   currentMessage.From.ID,
			ReceiverID: currentMessage.ReplyToMessage.From.ID,
			Error:      err.Error(),
		})
		return fmt.Errorf("error querying receiver restrictions: %w", err)
	}

	thresholdMessageLimit := 60 * time.Second
	if time.Since(lastMessageDateTime) < thresholdMessageLimit {
		err := SendMessageWithReply(chatId, replyToMessageId, "Whoops you are not allowed to give karma yet :(")
		if err != nil {
			_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
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
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   currentMessage.From.ID,
			ReceiverID: currentMessage.ReplyToMessage.From.ID,
			Error:      err.Error(),
		})
		return fmt.Errorf("error updating last_karma_given for sender: %w", err)
	}

	return nil
}
