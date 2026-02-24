package upal

import "fmt"

// NewPublishChannelID returns a unique identifier for a publish channel.
func NewPublishChannelID() string {
	return fmt.Sprintf("ch-%s", GenerateID("")[1:]) // strip leading "-"
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

// PublishChannel links a pipeline workflow to an external distribution target.
type PublishChannel struct {
	ID   string             `json:"id"`
	Name string             `json:"name"`
	Type PublishChannelType `json:"type"`
}
