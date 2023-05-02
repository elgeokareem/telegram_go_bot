package structs

type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	IsForum   *bool  `json:"is_forum,omitempty"`
	// Photo                              *ChatPhoto       `json:"photo,omitempty"`
	ActiveUsernames                    []string `json:"active_usernames,omitempty"`
	EmojiStatusCustomEmojiID           string   `json:"emoji_status_custom_emoji_id,omitempty"`
	Bio                                string   `json:"bio,omitempty"`
	HasPrivateForwards                 *bool    `json:"has_private_forwards,omitempty"`
	HasRestrictedVoiceAndVideoMessages *bool    `json:"has_restricted_voice_and_video_messages,omitempty"`
	JoinToSendMessages                 *bool    `json:"join_to_send_messages,omitempty"`
	JoinByRequest                      *bool    `json:"join_by_request,omitempty"`
	Description                        string   `json:"description,omitempty"`
	InviteLink                         string   `json:"invite_link,omitempty"`
	PinnedMessage                      *Message `json:"pinned_message,omitempty"`
	// Permissions                        *ChatPermissions `json:"permissions,omitempty"`
	SlowModeDelay                int           `json:"slow_mode_delay,omitempty"`
	MessageAutoDeleteTime        int           `json:"message_auto_delete_time,omitempty"`
	HasAggressiveAntiSpamEnabled *bool         `json:"has_aggressive_anti_spam_enabled,omitempty"`
	HasHiddenMembers             *bool         `json:"has_hidden_members,omitempty"`
	HasProtectedContent          *bool         `json:"has_protected_content,omitempty"`
	StickerSetName               string        `json:"sticker_set_name,omitempty"`
	CanSetStickerSet             *bool         `json:"can_set_sticker_set,omitempty"`
	LinkedChatID                 int64         `json:"linked_chat_id,omitempty"`
	Location                     *ChatLocation `json:"location,omitempty"`
}
