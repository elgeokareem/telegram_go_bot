package structs

type User struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name,omitempty"`
	Username                string `json:"username,omitempty"`
	LanguageCode            string `json:"language_code,omitempty"`
	IsPremium               *bool  `json:"is_premium,omitempty"`
	AddedToAttachmentMenu   *bool  `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups           *bool  `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages *bool  `json:"can_read_all_group_messages,omitempty"`
	SupportsInlineQueries   *bool  `json:"supports_inline_queries,omitempty"`
}
