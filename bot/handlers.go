package bot

import (
	"context"
	"fmt"
	"strings"
	"super-bot/core"

	"github.com/bwmarrin/discordgo"
)

// HandleStatusCommand handles the /status slash command
func HandleStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer response to avoid timeout
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Get dashboard data
	ctx := context.Background()
	data, err := core.GetDashboardData(ctx)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ Lỗi khi lấy dữ liệu: " + err.Error()),
		})
		return
	}

	// Create embed and components
	embed := CreateDashboardEmbed(data)
	components := CreateNodeButtons(data)

	// Send response
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}

// HandlePingCommand handles the /ping slash command
func HandlePingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	latency := s.HeartbeatLatency().Milliseconds()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🏓 Pong! Latency: %dms", latency),
		},
	})
}

// HandleButtonClick handles button interactions (node selection and refresh)
func HandleButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Defer response
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	ctx := context.Background()

	// Handle refresh button
	if customID == "refresh" {
		data, err := core.GetDashboardData(ctx)
		if err != nil {
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "❌ Lỗi khi refresh",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}

		embed := CreateDashboardEmbed(data)
		components := CreateNodeButtons(data)

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "🔄 Dashboard đã được cập nhật",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Handle node selection
	if strings.HasPrefix(customID, "node_") {
		nodeName := strings.TrimPrefix(customID, "node_")

		// Switch node
		err := core.SwitchNode(nodeName)
		if err != nil {
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "❌ Lỗi khi chuyển node",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}

		// Get updated data
		data, err := core.GetDashboardData(ctx)
		if err != nil {
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "❌ Lỗi khi lấy dữ liệu",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}

		embed := CreateDashboardEmbed(data)
		components := CreateNodeButtons(data)

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})

		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("✅ Đã chọn %s", nodeName),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
	}
}

// HandleStatusMessage handles the !status legacy command
func HandleStatusMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Get dashboard data
	ctx := context.Background()
	data, err := core.GetDashboardData(ctx)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Lỗi khi lấy dữ liệu: "+err.Error())
		return
	}

	// Create embed and components
	embed := CreateDashboardEmbed(data)
	components := CreateNodeButtons(data)

	// Send message
	msg := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}
	s.ChannelMessageSendComplex(m.ChannelID, msg)
}

// HandlePingMessage handles the !ping legacy command
func HandlePingMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	latency := s.HeartbeatLatency().Milliseconds()
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🏓 Pong! Latency: %dms", latency))
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
