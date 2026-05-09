package services

import (
	"bot/telegram/structs"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const setBirthdayCommand = "/set_birthday"
const birthdayReminderHourUTC = 13

func isSetBirthdayCommand(text string) bool {
	return isBotCommand(text, setBirthdayCommand)
}

func SetBirthdayFromCommand(conn *pgx.Conn, update structs.Update) error {
	message := update.Message
	if message == nil {
		return nil
	}

	chatID := message.Chat.ID
	if message.From == nil {
		return SendMessageWithReply(chatID, message.MessageID, "I need to know who is setting the birthday. Try again from a normal user account.")
	}

	if message.ReplyToMessage == nil || message.ReplyToMessage.From == nil {
		return SendMessageWithReply(chatID, message.MessageID, "Reply to someone's message with /set_birthday DD-MM-YYYY.")
	}

	birthday, err := parseBirthdayCommandDate(message.Text)
	if err != nil {
		return SendMessageWithReply(chatID, message.MessageID, "Use /set_birthday DD-MM-YYYY. Example: /set_birthday 24-12-1990")
	}

	targetUser := message.ReplyToMessage.From
	targetName := telegramUserDisplayName(targetUser)
	nextRunAt := nextBirthdayRunAt(birthday, time.Now().UTC())
	title := fmt.Sprintf("Celebrate %s's birthday! \U0001F382\U0001F389", targetName)
	description := fmt.Sprintf("Don't forget to wish %s a happy birthday!", targetName)
	dayBeforeMessage := fmt.Sprintf("Tomorrow is %s's birthday \U0001F382", targetName)
	dayOfMessage := fmt.Sprintf("Happy Birthday, %s!!! \U0001F382\U0001F389\U0001F382", targetName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin birthday transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var eventID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO events (
			chat_id,
			created_by_user_id,
			target_user_id,
			type,
			title,
			description,
			is_all_day,
			event_date,
			timezone,
			is_active
		) VALUES ($1,$2,$3,'birthday',$4,$5,TRUE,$6,'UTC',TRUE)
		RETURNING id
	`,
		chatID,
		message.From.ID,
		targetUser.ID,
		title,
		description,
		birthday.Format("2006-01-02"),
	).Scan(&eventID); err != nil {
		if isUniqueBirthdayConstraintError(err) {
			return SendMessageWithReply(
				chatID,
				message.MessageID,
				fmt.Sprintf("A birthday is already saved for this chat on %s.", birthday.Format("02-01-2006")),
			)
		}

		return fmt.Errorf("insert birthday event: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO event_recurrence (event_id, frequency, interval_value, next_run_at)
		VALUES ($1, 'yearly', 1, $2)
	`, eventID, nextRunAt); err != nil {
		return fmt.Errorf("insert birthday recurrence: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO event_reminders (event_id, offset_minutes, is_active, message_template)
		VALUES
			($1, -1440, TRUE, $2),
			($1, 0, TRUE, $3)
	`, eventID, dayBeforeMessage, dayOfMessage); err != nil {
		return fmt.Errorf("insert birthday reminders: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit birthday transaction: %w", err)
	}

	return SendMessageWithReply(
		chatID,
		message.MessageID,
		fmt.Sprintf(
			"Birthday event created \U00002705\nPerson: %s\nDate: %s\nReminder time: %02d:00 UTC\nEvent ID: %d\nI'll remind this chat every year.",
			targetName,
			birthday.Format("02-01-2006"),
			birthdayReminderHourUTC,
			eventID,
		),
	)
}

func parseBirthdayCommandDate(text string) (time.Time, error) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) != 2 {
		return time.Time{}, fmt.Errorf("expected command and date")
	}

	return time.Parse("02-01-2006", fields[1])
}

func isUniqueBirthdayConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_events_unique_birthday_chat_event_date"
}

func telegramUserDisplayName(user *structs.User) string {
	if user == nil {
		return "this person"
	}

	name := strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " "))
	if name != "" {
		return name
	}

	if strings.TrimSpace(user.Username) != "" {
		return "@" + strings.TrimSpace(user.Username)
	}

	return "this person"
}

func nextBirthdayRunAt(birthday time.Time, now time.Time) time.Time {
	candidate := birthdayOccurrenceInYear(birthday, now.Year())
	if !candidate.After(now) {
		candidate = birthdayOccurrenceInYear(birthday, now.Year()+1)
	}

	return candidate
}

func birthdayOccurrenceInYear(birthday time.Time, year int) time.Time {
	month := birthday.Month()
	day := birthday.Day()
	if month == time.February && day == 29 && !isLeapYear(year) {
		return time.Date(year, time.March, 1, birthdayReminderHourUTC, 0, 0, 0, time.UTC)
	}

	return time.Date(year, month, day, birthdayReminderHourUTC, 0, 0, 0, time.UTC)
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
