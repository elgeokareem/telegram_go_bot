package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type DueEventReminder struct {
	EventID         int64
	ReminderID      int64
	ChatID          int64
	Title           string
	Description     *string
	MessageTemplate *string
	ScheduledFor    time.Time
}

func ProcessDueEventReminders(ctx context.Context, conn *pgx.Conn) error {
	dueReminders, err := getDueEventReminders(ctx, conn)
	if err != nil {
		return err
	}

	for _, reminder := range dueReminders {
		message := buildReminderMessage(reminder)
		if err := SendMessage(reminder.ChatID, message); err != nil {
			if logErr := upsertEventDeliveryLog(ctx, conn, reminder, "failed", nil, err.Error()); logErr != nil {
				return fmt.Errorf("send reminder: %w; log failure: %w", err, logErr)
			}
			continue
		}

		sentAt := time.Now().UTC()
		if err := upsertEventDeliveryLog(ctx, conn, reminder, "sent", &sentAt, ""); err != nil {
			return err
		}
	}

	if err := closeCompletedEventOccurrences(ctx, conn); err != nil {
		return err
	}

	if err := advanceCompletedRecurringOccurrences(ctx, conn); err != nil {
		return err
	}

	return nil
}

func getDueEventReminders(ctx context.Context, conn *pgx.Conn) ([]DueEventReminder, error) {
	rows, err := conn.Query(ctx, `
		SELECT
			e.id,
			rem.id,
			e.chat_id,
			e.title,
			e.description,
			rem.message_template,
			(r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute')) AS scheduled_for
		FROM events e
		JOIN event_recurrence r ON r.event_id = e.id
		JOIN event_reminders rem ON rem.event_id = e.id
		WHERE e.is_active = TRUE
			AND rem.is_active = TRUE
			AND r.next_run_at IS NOT NULL
			AND (r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute')) <= NOW()
			AND NOT EXISTS (
				SELECT 1
				FROM event_delivery_log log
				WHERE log.event_id = e.id
					AND log.reminder_id = rem.id
					AND log.scheduled_for = (r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute'))
					AND log.status = 'sent'
			)
		ORDER BY scheduled_for ASC
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("query due event reminders: %w", err)
	}
	defer rows.Close()

	reminders := make([]DueEventReminder, 0)
	for rows.Next() {
		var reminder DueEventReminder
		if err := rows.Scan(
			&reminder.EventID,
			&reminder.ReminderID,
			&reminder.ChatID,
			&reminder.Title,
			&reminder.Description,
			&reminder.MessageTemplate,
			&reminder.ScheduledFor,
		); err != nil {
			return nil, fmt.Errorf("scan due event reminder: %w", err)
		}
		reminders = append(reminders, reminder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due event reminders: %w", err)
	}

	return reminders, nil
}

func upsertEventDeliveryLog(ctx context.Context, conn *pgx.Conn, reminder DueEventReminder, status string, sentAt *time.Time, errorMessage string) error {
	var normalizedError *string
	if strings.TrimSpace(errorMessage) != "" {
		normalizedError = &errorMessage
	}

	_, err := conn.Exec(ctx, `
		INSERT INTO event_delivery_log (event_id, reminder_id, scheduled_for, sent_at, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id, COALESCE(reminder_id, 0), scheduled_for)
		DO UPDATE SET
			sent_at = EXCLUDED.sent_at,
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			updated_at = CURRENT_TIMESTAMP
	`, reminder.EventID, reminder.ReminderID, reminder.ScheduledFor, sentAt, status, normalizedError)
	if err != nil {
		return fmt.Errorf("upsert event delivery log: %w", err)
	}

	return nil
}

func closeCompletedEventOccurrences(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		WITH eligible AS (
			SELECT e.id, r.id AS recurrence_id
			FROM events e
			JOIN event_recurrence r ON r.event_id = e.id
			WHERE e.is_active = TRUE
				AND r.next_run_at IS NOT NULL
				AND r.next_run_at <= NOW()
				AND (r.frequency = 'none' OR r.occurrence_count = 1)
				AND NOT EXISTS (
					SELECT 1
					FROM event_reminders rem
					WHERE rem.event_id = e.id
						AND rem.is_active = TRUE
						AND (r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute')) > NOW()
				)
				AND NOT EXISTS (
					SELECT 1
					FROM event_reminders rem
					WHERE rem.event_id = e.id
						AND rem.is_active = TRUE
						AND NOT EXISTS (
							SELECT 1
							FROM event_delivery_log log
							WHERE log.event_id = e.id
								AND log.reminder_id = rem.id
								AND log.scheduled_for = (r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute'))
								AND log.status = 'sent'
						)
				)
		)
		UPDATE events e
		SET is_active = FALSE,
			updated_at = CURRENT_TIMESTAMP
		FROM eligible
		WHERE e.id = eligible.id
	`)
	if err != nil {
		return fmt.Errorf("close completed event occurrences: %w", err)
	}

	_, err = conn.Exec(ctx, `
		UPDATE event_recurrence r
		SET next_run_at = NULL,
			updated_at = CURRENT_TIMESTAMP
		FROM events e
		WHERE r.event_id = e.id
			AND e.is_active = FALSE
			AND r.next_run_at IS NOT NULL
			AND (r.frequency = 'none' OR r.occurrence_count = 1)
	`)
	if err != nil {
		return fmt.Errorf("clear completed recurrence next_run_at: %w", err)
	}

	return nil
}

func advanceCompletedRecurringOccurrences(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		UPDATE event_recurrence r
		SET next_run_at = CASE r.frequency
				WHEN 'daily' THEN r.next_run_at + make_interval(days => r.interval_value)
				WHEN 'weekly' THEN r.next_run_at + make_interval(weeks => r.interval_value)
				WHEN 'monthly' THEN r.next_run_at + make_interval(months => r.interval_value)
				WHEN 'yearly' THEN r.next_run_at + make_interval(years => r.interval_value)
				ELSE r.next_run_at
			END,
			occurrence_count = CASE
				WHEN r.occurrence_count IS NULL THEN NULL
				ELSE r.occurrence_count - 1
			END,
			updated_at = CURRENT_TIMESTAMP
		FROM events e
		WHERE r.event_id = e.id
			AND e.is_active = TRUE
			AND r.frequency <> 'none'
			AND r.next_run_at IS NOT NULL
			AND r.next_run_at <= NOW()
			AND (r.occurrence_count IS NULL OR r.occurrence_count > 1)
			AND NOT EXISTS (
				SELECT 1
				FROM event_reminders rem
				WHERE rem.event_id = e.id
					AND rem.is_active = TRUE
					AND (r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute')) > NOW()
			)
			AND NOT EXISTS (
				SELECT 1
				FROM event_reminders rem
				WHERE rem.event_id = e.id
					AND rem.is_active = TRUE
					AND NOT EXISTS (
						SELECT 1
						FROM event_delivery_log log
						WHERE log.event_id = e.id
							AND log.reminder_id = rem.id
							AND log.scheduled_for = (r.next_run_at + (rem.offset_minutes * INTERVAL '1 minute'))
							AND log.status = 'sent'
					)
			)
	`)
	if err != nil {
		return fmt.Errorf("advance completed recurring occurrences: %w", err)
	}

	return nil
}

func buildReminderMessage(reminder DueEventReminder) string {
	if reminder.MessageTemplate != nil && strings.TrimSpace(*reminder.MessageTemplate) != "" {
		return strings.TrimSpace(*reminder.MessageTemplate)
	}

	var b strings.Builder
	b.WriteString("Reminder: ")
	b.WriteString(strings.TrimSpace(reminder.Title))

	if reminder.Description != nil && strings.TrimSpace(*reminder.Description) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(*reminder.Description))
	}

	return b.String()
}
