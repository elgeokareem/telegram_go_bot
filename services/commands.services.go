package services

import "strings"

func isBotCommand(text string, command string) bool {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return false
	}

	messageCommand := strings.Split(fields[0], "@")[0]
	return messageCommand == command
}

func SendCommandsHelp(chatID int64) error {
	return SendMessage(chatID, strings.TrimSpace(`Available bot commands:

/command
Shows this help message with details for every command.

/new_event
Opens the event Web App. Use it to create custom events, reminders, or birthdays with a form.

/ask_catholic_church <question>
Asks Magisterium AI a question about Catholic teaching and replies with the answer.
Example: /ask_catholic_church What does the Church teach about forgiveness?

/set_birthday DD-MM-YYYY
Creates a yearly birthday event. Use it as a reply to the person's message so the bot knows whose birthday to save.
Example: reply to Maria and send /set_birthday 24-12-1990

/show_events
Shows all active events in this group with their IDs, types, titles, and dates.

/delete_event <id>
Deletes an event by its ID. Only group admins can use this command.
Example: /delete_event 42

/lovedusers
Shows the users with the most positive karma in this chat.

/hatedusers
Shows the users with the most negative karma in this chat.`))
}
