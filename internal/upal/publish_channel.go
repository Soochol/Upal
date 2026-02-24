package upal

// NewPublishChannelID returns a unique identifier for a publish channel.
func NewPublishChannelID() string {
	return GenerateID("ch")
}

// PublishChannelType identifies the external platform for content distribution.
type PublishChannelType string

const (
	ChannelWordPress PublishChannelType = "wordpress"
	ChannelYouTube   PublishChannelType = "youtube"
	ChannelSlack     PublishChannelType = "slack"
	ChannelTelegram  PublishChannelType = "telegram"
	ChannelSubstack  PublishChannelType = "substack"
	ChannelDiscord   PublishChannelType = "discord"
	ChannelMedium    PublishChannelType = "medium"
	ChannelTikTok    PublishChannelType = "tiktok"
	ChannelHTTP      PublishChannelType = "http"
)

// ValidChannelType reports whether t is a known publish channel type.
func ValidChannelType(t PublishChannelType) bool {
	switch t {
	case ChannelWordPress, ChannelYouTube, ChannelSlack, ChannelTelegram,
		ChannelSubstack, ChannelDiscord, ChannelMedium, ChannelTikTok, ChannelHTTP:
		return true
	}
	return false
}

// PublishChannel links a pipeline workflow to an external distribution target.
type PublishChannel struct {
	ID   string             `json:"id"`
	Name string             `json:"name"`
	Type PublishChannelType `json:"type"`
}
