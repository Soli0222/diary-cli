package models

import "time"

// Note represents a Misskey note
type Note struct {
	ID         string     `json:"id"`
	CreatedAt  time.Time  `json:"createdAt"`
	Text       *string    `json:"text"`
	CW         *string    `json:"cw"`
	UserID     string     `json:"userId"`
	User       *UserLite  `json:"user,omitempty"`
	ReplyID    *string    `json:"replyId"`
	RenoteID   *string    `json:"renoteId"`
	Renote     *Note      `json:"renote,omitempty"`
	Reply      *Note      `json:"reply,omitempty"`
	Visibility string     `json:"visibility"`
	LocalOnly  bool       `json:"localOnly"`
	IsHidden   bool       `json:"isHidden"`
	Tags       []string   `json:"tags"`
	FileIDs    []string   `json:"fileIds"`
	ChannelID  *string    `json:"channelId"`
}

// UserLite represents a minimal Misskey user
type UserLite struct {
	ID       string  `json:"id"`
	Name     *string `json:"name"`
	Username string  `json:"username"`
	Host     *string `json:"host"`
}

// MeDetailed represents the authenticated user's detailed info
type MeDetailed struct {
	ID       string  `json:"id"`
	Name     *string `json:"name"`
	Username string  `json:"username"`
}

// GetDisplayText returns the text content of a note, handling CW and renotes
func (n *Note) GetDisplayText() string {
	text := ""

	if n.CW != nil {
		text += "[CW: " + *n.CW + "] "
	}

	if n.Text != nil {
		text += *n.Text
	}

	// If this is a renote with no text, show the renoted content
	if n.Text == nil && n.Renote != nil {
		if n.Renote.Text != nil {
			text = "[RN] " + *n.Renote.Text
		}
	}

	return text
}

// IsOriginalNote returns true if this note is not a pure renote
func (n *Note) IsOriginalNote() bool {
	// Pure renote has no text and only renoteId
	return n.Text != nil || n.RenoteID == nil
}
