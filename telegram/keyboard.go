package telegram

import (
	"fmt"
	"strings"
	"super-bot/core"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CreateNodeKeyboard creates the inline keyboard with node selection buttons
func CreateNodeKeyboard(data *core.DashboardData) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	// Create node buttons
	for _, node := range data.Singbox.AllNodes {
		// Display logic
		display := strings.Replace(node, "WG-Solid-", "", -1)
		display = strings.Replace(display, "WG-", "", -1)

		// Status icon
		icon := "🌐"
		if node == data.Singbox.CurrentNode {
			icon = "🟢"
		}

		// Delay
		delay := data.Singbox.NodeDelays[node]
		delayStr := "N/A"
		if delay > 0 {
			delayStr = fmt.Sprintf("%dms", delay)
		}

		label := fmt.Sprintf("%s %s (%s)", icon, display, delayStr)
		btn := tgbotapi.NewInlineKeyboardButtonData(label, "set|"+node)

		currentRow = append(currentRow, btn)

		// 2 buttons per row
		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	// Add remaining button if odd number
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// Add Refresh button
	refreshBtn := tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Dashboard", "refresh")
	rows = append(rows, []tgbotapi.InlineKeyboardButton{refreshBtn})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
