package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"bot/telegram/structs"

	"github.com/jackc/pgx/v5"
)

// HandleAddCommand handles /add command in group chat - shows the birthday button
func HandleAddCommand(chatID int64) error {
	return SendBirthdayButton(chatID, GetBotUsername())
}

// HandleBirthdayCommand handles /birthday command in group chat (alias for /add)
func HandleBirthdayCommand(chatID int64) error {
	return SendBirthdayButton(chatID, GetBotUsername())
}

// HandleStartCommand handles /start command with potential deep link parameters
// This starts the conversational flow when user comes from a group
func HandleStartCommand(conn *pgx.Conn, chatID int64, user *structs.User, text string) (bool, error) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		// No parameter, just a regular /start - send welcome message
		return false, nil
	}

	param := parts[1]
	paramType, groupID, err := ParseStartParameter(param)
	if err != nil {
		return false, nil
	}

	if paramType == "birthday" {
		// User came from birthday button - start conversational flow
		Conversations.StartConversation(user.ID, groupID, StateAwaitingDate)

		msg := "🎂 *¡Hola!* Vamos a registrar tu cumpleaños.\n\n" +
			"📅 *¿Cuál es tu fecha de cumpleaños?*\n\n" +
			"Escribe la fecha en formato DD/MM (ejemplo: `25/12`)"

		return true, SendMessage(chatID, msg)
	}

	return false, nil
}

// HandlePrivateMessage handles messages in private chat (for conversational flow)
func HandlePrivateMessage(conn *pgx.Conn, chatID int64, user *structs.User, text string) (bool, error) {
	conv := Conversations.GetConversation(user.ID)
	if conv == nil {
		return false, nil // No active conversation
	}

	switch conv.State {
	case StateAwaitingDate:
		return handleAwaitingDate(conn, chatID, user, text, conv)
	case StateAwaitingConfirm:
		return handleAwaitingConfirm(conn, chatID, user, text, conv)
	}

	return false, nil
}

func handleAwaitingDate(conn *pgx.Conn, chatID int64, user *structs.User, text string, conv *ConversationState) (bool, error) {
	day, month, err := ParseBirthdayDate(text)
	if err != nil {
		msg := "❌ Formato inválido. Por favor escribe tu fecha de cumpleaños en formato DD/MM\n\n" +
			"Ejemplo: `25/12` para el 25 de diciembre"
		return true, SendMessage(chatID, msg)
	}

	// Store the date and ask for confirmation
	Conversations.SetData(user.ID, "day", strconv.Itoa(day))
	Conversations.SetData(user.ID, "month", strconv.Itoa(month))
	Conversations.UpdateState(user.ID, StateAwaitingConfirm)

	monthNames := []string{
		"", "enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	}

	msg := fmt.Sprintf("📆 Tu cumpleaños es el *%d de %s*\n\n"+
		"¿Es correcto?\n\n"+
		"Escribe *sí* para confirmar o *no* para corregir", day, monthNames[month])

	return true, SendMessage(chatID, msg)
}

func handleAwaitingConfirm(conn *pgx.Conn, chatID int64, user *structs.User, text string, conv *ConversationState) (bool, error) {
	text = strings.ToLower(strings.TrimSpace(text))

	if text == "no" || text == "n" || text == "corregir" {
		// Go back to date input
		Conversations.UpdateState(user.ID, StateAwaitingDate)
		msg := "📅 *¿Cuál es tu fecha de cumpleaños?*\n\n" +
			"Escribe la fecha en formato DD/MM (ejemplo: `25/12`)"
		return true, SendMessage(chatID, msg)
	}

	if text == "sí" || text == "si" || text == "s" || text == "yes" || text == "y" || text == "confirmar" {
		// Save the birthday
		dayStr := conv.Data["day"]
		monthStr := conv.Data["month"]
		day, _ := strconv.Atoi(dayStr)
		month, _ := strconv.Atoi(monthStr)
		groupID := conv.GroupID

		// Ensure user exists in users_ranking for the group
		sql := `
			INSERT INTO users_ranking (user_id, group_id, first_name, last_name, username, karma)
			VALUES ($1, $2, $3, $4, $5, 0)
			ON CONFLICT (user_id, group_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				username = EXCLUDED.username
		`
		_, err := conn.Exec(context.Background(), sql, user.ID, groupID, user.FirstName, user.LastName, user.Username)
		if err != nil {
			fmt.Printf("Warning: Failed to ensure user in ranking: %v\n", err)
		}

		err = CreateBirthdayEvent(conn, user.ID, groupID, day, month)
		if err != nil {
			Conversations.EndConversation(user.ID)
			return true, SendMessage(chatID, fmt.Sprintf("❌ Error al registrar cumpleaños: %v", err))
		}

		Conversations.EndConversation(user.ID)

		monthNames := []string{
			"", "enero", "febrero", "marzo", "abril", "mayo", "junio",
			"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
		}

		msg := fmt.Sprintf("🎉 *¡Listo!* Tu cumpleaños ha sido registrado.\n\n"+
			"📅 El *%d de %s* recibirás una felicitación especial en el grupo.\n\n"+
			"¡Nos vemos! 👋", day, monthNames[month])

		return true, SendMessage(chatID, msg)
	}

	// Invalid response
	msg := "Por favor escribe *sí* para confirmar o *no* para corregir la fecha."
	return true, SendMessage(chatID, msg)
}

// HandleCreateEventCommand handles /createevent command in group chat
func HandleCreateEventCommand(conn *pgx.Conn, chatID, userID int64, text string) error {
	title, description, eventDate, isRecurring, recurrenceType, recurrenceDay, err := ParseEventCommand(text)
	if err != nil {
		return SendMessage(chatID, fmt.Sprintf("❌ %v\n\nEjemplos:\n• `/createevent Reunión | Discutir proyecto | 15/01/2025 14:00`\n• `/createevent Sync | Semanal | weekly MON 09:00`", err))
	}

	id, err := CreateEvent(conn, chatID, userID, title, description, eventDate, isRecurring, recurrenceType, recurrenceDay)
	if err != nil {
		return SendMessage(chatID, fmt.Sprintf("❌ Error al crear evento: %v", err))
	}

	var dateInfo string
	if isRecurring {
		dateInfo = fmt.Sprintf("%s %s", recurrenceType, recurrenceDay)
	} else if eventDate != nil {
		dateInfo = eventDate.Format("02/01/2006 15:04")
	}

	return SendMessage(chatID, fmt.Sprintf("✅ *Evento creado* (ID: %d)\n\n📌 *%s*\n📝 %s\n📅 %s", id, title, description, dateInfo))
}

// HandleEventsCommand handles /events command in group chat
func HandleEventsCommand(conn *pgx.Conn, chatID int64) error {
	events, err := GetGroupEvents(conn, chatID)
	if err != nil {
		return SendMessage(chatID, fmt.Sprintf("❌ Error al obtener eventos: %v", err))
	}

	if len(events) == 0 {
		return SendMessage(chatID, "📅 No hay eventos registrados en este grupo.")
	}

	var sb strings.Builder
	sb.WriteString("📅 *Eventos del grupo:*\n\n")

	for _, event := range events {
		emoji := "📌"
		if event.EventType == structs.EventTypeBirthday {
			emoji = "🎂"
		}

		var dateInfo string
		if event.IsRecurring {
			dateInfo = fmt.Sprintf("%s %s", event.RecurrenceType, event.RecurrenceDay)
		} else if event.EventDate != nil {
			dateInfo = event.EventDate.Format("02/01/2006 15:04")
		}

		userName := event.FirstName
		if userName == "" {
			userName = fmt.Sprintf("User %d", event.UserID)
		}

		sb.WriteString(fmt.Sprintf("%s *%s* (ID: %d)\n", emoji, event.Title, event.ID))
		if event.Description != "" {
			sb.WriteString(fmt.Sprintf("   📝 %s\n", event.Description))
		}
		sb.WriteString(fmt.Sprintf("   📅 %s | 👤 %s\n\n", dateInfo, userName))
	}

	return SendMessage(chatID, sb.String())
}

// HandleDeleteEventCommand handles /deleteevent <id> command in group chat
func HandleDeleteEventCommand(conn *pgx.Conn, chatID int64, text string) error {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return SendMessage(chatID, "❌ Uso: /deleteevent <id>")
	}

	eventID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return SendMessage(chatID, "❌ ID de evento inválido")
	}

	err = DeleteEvent(conn, eventID, chatID)
	if err != nil {
		return SendMessage(chatID, fmt.Sprintf("❌ Error al eliminar evento: %v", err))
	}

	return SendMessage(chatID, fmt.Sprintf("✅ Evento %d eliminado", eventID))
}

// IsPrivateChat checks if the chat is a private chat
func IsPrivateChat(chatType string) bool {
	return chatType == "private"
}

// IsGroupChat checks if the chat is a group or supergroup
func IsGroupChat(chatType string) bool {
	return chatType == "group" || chatType == "supergroup"
}
