package bot

import (
	"fmt"
	"strings"
	"super-bot/core"

	"github.com/bwmarrin/discordgo"
)

// CreateNodeButtons creates interactive buttons for VPN node selection
func CreateNodeButtons(data *core.DashboardData) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	// Add node buttons (max 24 buttons to allow refresh button as the 25th)
	// Discord allows max 5 Action Rows per message, max 5 buttons per row
	for i, node := range data.Singbox.AllNodes {
		if i >= 24 {
			break
		}

		// Shorten display name
		display := strings.ReplaceAll(node, "WG-Solid-", "")
		display = strings.ReplaceAll(display, "WG-", "")

		// Get delay
		delay := data.Singbox.NodeDelays[node]
		delayStr := "N/A"
		if delay > 0 {
			delayStr = fmt.Sprintf("%dms", delay)
		}

		// Determine style and emoji
		style := discordgo.SecondaryButton
		emoji := "🌐"
		if node == data.Singbox.CurrentNode {
			style = discordgo.SuccessButton
			emoji = "🟢"
		}

		button := discordgo.Button{
			Label:    fmt.Sprintf("%s (%s)", display, delayStr),
			Style:    style,
			CustomID: "node_" + node,
			Emoji: &discordgo.ComponentEmoji{
				Name: emoji,
			},
		}

		currentRow = append(currentRow, button)

		// Discord allows max 5 buttons per row
		if len(currentRow) == 5 {
			components = append(components, discordgo.ActionsRow{
				Components: currentRow,
			})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	// Add refresh button to current row or new row
	refreshButton := discordgo.Button{
		Label:    "Refresh",
		Style:    discordgo.PrimaryButton,
		CustomID: "refresh",
		Emoji: &discordgo.ComponentEmoji{
			Name: "🔄",
		},
	}

	currentRow = append(currentRow, refreshButton)

	// Add final row
	if len(currentRow) > 0 {
		components = append(components, discordgo.ActionsRow{
			Components: currentRow,
		})
	}

	return components
}
