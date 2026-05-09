package services

import (
	"bot/telegram/config"
	"bot/telegram/shared"
	"bot/telegram/structs"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const askCatholicChurchCommand = "/ask_catholic_church"

type magisteriumChatRequest struct {
	Model    string               `json:"model"`
	Messages []magisteriumMessage `json:"messages"`
	Stream   bool                 `json:"stream"`
}

type magisteriumMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type magisteriumChatResponse struct {
	Choices []struct {
		Message magisteriumMessage `json:"message"`
	} `json:"choices"`
	Citations []magisteriumCitation `json:"citations"`
}

type magisteriumCitation struct {
	DocumentTitle     string `json:"document_title"`
	DocumentAuthor    string `json:"document_author"`
	DocumentReference string `json:"document_reference"`
	SourceURL         string `json:"source_url"`
}

func isAskCatholicChurchCommand(text string) bool {
	return isBotCommand(text, askCatholicChurchCommand)
}

func AskCatholicChurchFromCommand(update structs.Update) error {
	message := update.Message
	if message == nil {
		return nil
	}

	chatID := message.Chat.ID
	question := commandArgument(message.Text)
	if question == "" {
		return SendMessageWithReply(chatID, message.MessageID, "Use /ask_catholic_church followed by your question. Example: /ask_catholic_church What does the Church teach about forgiveness?")
	}

	answer, err := askMagisterium(question)
	if err != nil {
		if replyErr := SendMessageWithReply(chatID, message.MessageID, "I couldn't get an answer from Magisterium AI right now. Please try again later."); replyErr != nil {
			return fmt.Errorf("ask magisterium: %w; send error reply: %w", err, replyErr)
		}

		return fmt.Errorf("ask magisterium: %w", err)
	}

	return SendLongHTMLMessageWithReply(chatID, message.MessageID, answer)
}

func askMagisterium(question string) (string, error) {
	env := config.Current
	if env.MagisteriumAPIKey == "" {
		return "", fmt.Errorf("missing MAGISTERIUM_API_KEY")
	}

	requestBody := magisteriumChatRequest{
		Model: "magisterium-1",
		Messages: []magisteriumMessage{{
			Role:    "user",
			Content: question,
		}},
		Stream: false,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, env.MagisteriumAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+env.MagisteriumAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := shared.CustomClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("magisterium API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	var chatResponse magisteriumChatResponse
	if err := json.Unmarshal(responseBody, &chatResponse); err != nil {
		return "", err
	}

	if len(chatResponse.Choices) == 0 || strings.TrimSpace(chatResponse.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("magisterium API returned no answer")
	}

	return formatMagisteriumAnswer(chatResponse), nil
}

func formatMagisteriumAnswer(response magisteriumChatResponse) string {
	answer := strings.TrimSpace(response.Choices[0].Message.Content)
	if len(response.Citations) == 0 {
		return answer
	}

	var b strings.Builder
	b.WriteString(answer)
	b.WriteString("\n\nSources:")

	for i, citation := range response.Citations {
		if i >= 3 {
			break
		}

		title := strings.TrimSpace(citation.DocumentTitle)
		if title == "" {
			title = "Catholic source"
		}

		b.WriteString(fmt.Sprintf("\n%d. %s", i+1, title))
		if strings.TrimSpace(citation.DocumentReference) != "" {
			b.WriteString(" ")
			b.WriteString(strings.TrimSpace(citation.DocumentReference))
		}
		if strings.TrimSpace(citation.SourceURL) != "" {
			b.WriteString(" - ")
			b.WriteString(strings.TrimSpace(citation.SourceURL))
		}
	}

	return b.String()
}

func commandArgument(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) < 2 {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), fields[0]))
}
