package services

import (
	"sync"
	"time"
)

// ConversationState represents the current state of a user's conversation with the bot
type ConversationState struct {
	UserID    int64
	GroupID   int64             // The group this conversation is related to
	State     string            // Current state: "awaiting_date", "awaiting_confirm", etc.
	Data      map[string]string // Temporary data storage
	ExpiresAt time.Time
}

// ConversationManager manages active conversations
type ConversationManager struct {
	mu            sync.RWMutex
	conversations map[int64]*ConversationState // keyed by userID
}

// Global conversation manager
var Conversations = &ConversationManager{
	conversations: make(map[int64]*ConversationState),
}

// Conversation states
const (
	StateNone            = ""
	StateAwaitingDate    = "awaiting_birthday_date"
	StateAwaitingConfirm = "awaiting_confirm"
)

// StartConversation starts a new conversation for a user
func (cm *ConversationManager) StartConversation(userID, groupID int64, state string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.conversations[userID] = &ConversationState{
		UserID:    userID,
		GroupID:   groupID,
		State:     state,
		Data:      make(map[string]string),
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10 minute timeout
	}
}

// GetConversation gets the current conversation state for a user
func (cm *ConversationManager) GetConversation(userID int64) *ConversationState {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conv, exists := cm.conversations[userID]
	if !exists || time.Now().After(conv.ExpiresAt) {
		return nil
	}
	return conv
}

// UpdateState updates the conversation state
func (cm *ConversationManager) UpdateState(userID int64, state string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conv, exists := cm.conversations[userID]; exists {
		conv.State = state
		conv.ExpiresAt = time.Now().Add(10 * time.Minute)
	}
}

// SetData stores temporary data in the conversation
func (cm *ConversationManager) SetData(userID int64, key, value string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conv, exists := cm.conversations[userID]; exists {
		conv.Data[key] = value
	}
}

// EndConversation ends a conversation
func (cm *ConversationManager) EndConversation(userID int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.conversations, userID)
}
