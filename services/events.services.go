package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bot/telegram/structs"

	"github.com/jackc/pgx/v5"
)

// CreateBirthdayEvent registers a user's birthday for a specific group
func CreateBirthdayEvent(conn *pgx.Conn, userID, groupID int64, day, month int) error {
	recurrenceDay := fmt.Sprintf("%02d/%02d", day, month)

	sql := `
		INSERT INTO group_events (group_id, user_id, event_type, title, is_recurring, recurrence_type, recurrence_day)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, group_id, event_type)
		DO UPDATE SET recurrence_day = $7
	`

	_, err := conn.Exec(context.Background(), sql,
		groupID,
		userID,
		structs.EventTypeBirthday,
		"Cumpleaños",
		true,
		structs.RecurrenceAnnually,
		recurrenceDay,
	)

	return err
}

// CreateEvent creates a new event (one-time or recurring)
func CreateEvent(conn *pgx.Conn, groupID, userID int64, title, description string, eventDate *time.Time, isRecurring bool, recurrenceType, recurrenceDay string) (int64, error) {
	sql := `
		INSERT INTO group_events (group_id, user_id, event_type, title, description, event_date, is_recurring, recurrence_type, recurrence_day)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var id int64
	err := conn.QueryRow(context.Background(), sql,
		groupID,
		userID,
		structs.EventTypeEvent,
		title,
		description,
		eventDate,
		isRecurring,
		recurrenceType,
		recurrenceDay,
	).Scan(&id)

	return id, err
}

// GetGroupEvents returns all events for a group
func GetGroupEvents(conn *pgx.Conn, groupID int64) ([]structs.EventWithUser, error) {
	sql := `
		SELECT e.id, e.group_id, e.user_id, e.event_type, e.title, e.description, 
		       e.event_date, e.is_recurring, e.recurrence_type, e.recurrence_day, e.created_at,
		       COALESCE(u.first_name, '') as first_name, 
		       COALESCE(u.last_name, '') as last_name, 
		       COALESCE(u.username, '') as username
		FROM group_events e
		LEFT JOIN users_ranking u ON e.user_id = u.user_id AND e.group_id = u.group_id
		WHERE e.group_id = $1
		ORDER BY e.created_at DESC
	`

	rows, err := conn.Query(context.Background(), sql, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []structs.EventWithUser
	for rows.Next() {
		var event structs.EventWithUser
		err := rows.Scan(
			&event.ID, &event.GroupID, &event.UserID, &event.EventType, &event.Title, &event.Description,
			&event.EventDate, &event.IsRecurring, &event.RecurrenceType, &event.RecurrenceDay, &event.CreatedAt,
			&event.FirstName, &event.LastName, &event.Username,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetTodaysBirthdays returns all birthdays that should be announced today
func GetTodaysBirthdays(conn *pgx.Conn) ([]structs.EventWithUser, error) {
	now := time.Now()
	todayPattern := fmt.Sprintf("%02d/%02d", now.Day(), int(now.Month()))

	sql := `
		SELECT e.id, e.group_id, e.user_id, e.event_type, e.title, e.description, 
		       e.event_date, e.is_recurring, e.recurrence_type, e.recurrence_day, e.created_at,
		       COALESCE(u.first_name, '') as first_name, 
		       COALESCE(u.last_name, '') as last_name, 
		       COALESCE(u.username, '') as username
		FROM group_events e
		LEFT JOIN users_ranking u ON e.user_id = u.user_id AND e.group_id = u.group_id
		WHERE e.event_type = $1 AND e.recurrence_day = $2
	`

	rows, err := conn.Query(context.Background(), sql, structs.EventTypeBirthday, todayPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var birthdays []structs.EventWithUser
	for rows.Next() {
		var event structs.EventWithUser
		err := rows.Scan(
			&event.ID, &event.GroupID, &event.UserID, &event.EventType, &event.Title, &event.Description,
			&event.EventDate, &event.IsRecurring, &event.RecurrenceType, &event.RecurrenceDay, &event.CreatedAt,
			&event.FirstName, &event.LastName, &event.Username,
		)
		if err != nil {
			return nil, err
		}
		birthdays = append(birthdays, event)
	}

	return birthdays, rows.Err()
}

// DeleteEvent removes an event by ID
func DeleteEvent(conn *pgx.Conn, eventID, groupID int64) error {
	sql := `DELETE FROM group_events WHERE id = $1 AND group_id = $2`
	result, err := conn.Exec(context.Background(), sql, eventID, groupID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("event not found")
	}

	return nil
}

// ParseBirthdayDate parses DD/MM format into day and month
func ParseBirthdayDate(dateStr string) (int, int, error) {
	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid date format, use DD/MM")
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("invalid day")
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("invalid month")
	}

	return day, month, nil
}

// ParseEventCommand parses the /createevent command
// Format: /createevent title | description | DD/MM/YYYY HH:MM
// Or:     /createevent title | description | weekly MON 09:00
func ParseEventCommand(text string) (title, description string, eventDate *time.Time, isRecurring bool, recurrenceType, recurrenceDay string, err error) {
	// Remove the command prefix
	text = strings.TrimPrefix(text, "/createevent")
	text = strings.TrimPrefix(text, "@WillibertoBot")
	text = strings.TrimSpace(text)

	parts := strings.Split(text, "|")
	if len(parts) < 3 {
		return "", "", nil, false, "", "", fmt.Errorf("formato: /createevent título | descripción | fecha")
	}

	title = strings.TrimSpace(parts[0])
	description = strings.TrimSpace(parts[1])
	dateStr := strings.TrimSpace(parts[2])

	// Check if it's a recurring event
	lowerDate := strings.ToLower(dateStr)
	if strings.HasPrefix(lowerDate, "weekly") || strings.HasPrefix(lowerDate, "monthly") || strings.HasPrefix(lowerDate, "annually") {
		isRecurring = true
		dateParts := strings.Fields(dateStr)
		recurrenceType = strings.ToLower(dateParts[0])
		recurrenceDay = strings.Join(dateParts[1:], " ")
		return title, description, nil, isRecurring, recurrenceType, recurrenceDay, nil
	}

	// Parse one-time event date
	parsedDate, parseErr := time.Parse("02/01/2006 15:04", dateStr)
	if parseErr != nil {
		return "", "", nil, false, "", "", fmt.Errorf("formato de fecha inválido, use DD/MM/YYYY HH:MM")
	}
	eventDate = &parsedDate

	return title, description, eventDate, false, "", "", nil
}

// GetBirthdayAnnouncementMessage returns a random celebratory message
func GetBirthdayAnnouncementMessage(firstName, lastName string) string {
	fullName := strings.TrimSpace(firstName + " " + lastName)
	if fullName == "" {
		fullName = "Alguien especial"
	}

	messages := []string{
		fmt.Sprintf("🎉🎂 *¡Feliz cumpleaños, %s!* 🎂🎉\n\n¡Que este día esté lleno de alegría y momentos especiales! 🥳", fullName),
		fmt.Sprintf("🌟 *¡El día ha llegado!* 🎈\n\n*%s* está celebrando su cumpleaños hoy. ¡Muchas felicidades! 🎁", fullName),
		fmt.Sprintf("🎊 *¡Hoy es el día especial de %s!* 🎂\n\n¡Que todos tus deseos se hagan realidad! ✨", fullName),
	}

	// Simple rotation based on current second
	idx := time.Now().Second() % len(messages)
	return messages[idx]
}
