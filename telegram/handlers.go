package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"super-bot/core"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleStatusCommand handles the /status command
func HandleStatusCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🔄 *Đang tải dữ liệu...*")
	msg.ParseMode = "Markdown"
	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Println("Error sending initial message:", err)
		return
	}

	// Fetch data
	data, err := core.GetDashboardData(context.Background())
	if err != nil {
		edit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, sentMsg.MessageID, "❌ Lỗi khi tải dữ liệu: "+err.Error())
		bot.Send(edit)
		return
	}

	// Update message with dashboard
	text := FormatDashboardMessage(data)
	keyboard := CreateNodeKeyboard(data)

	edit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, sentMsg.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard

	bot.Send(edit)
}

// HandleButtonCallback handles button clicks (node selection and refresh)
func HandleButtonCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if !IsAuthorizedCallback(update) {
		answerUnauthorized(bot, update)
		return
	}

	callback := update.CallbackQuery
	data := callback.Data

	// Verify authorization? (Assuming handled by main loop filter, but good to have)
	// For now we trust the caller if they can see the button

	if data == "refresh" {
		// Answer callback immediately
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Đang cập nhật..."))
	} else if strings.HasPrefix(data, "set|") {
		nodeName := strings.TrimPrefix(data, "set|")

		// Switch node
		err := core.SwitchNode(nodeName)
		if err != nil {
			bot.Request(tgbotapi.NewCallbackWithAlert(callback.ID, "❌ Lỗi: "+err.Error()))
			return // Don't refresh if failed
		}

		bot.Request(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("✅ Đã chọn %s", nodeName)))
	}

	// Refresh dashboard
	dashData, err := core.GetDashboardData(context.Background())
	if err != nil {
		// If fails, we can't update dashboard, just log it
		log.Println("Error updating dashboard:", err)
		return
	}

	text := FormatDashboardMessage(dashData)
	keyboard := CreateNodeKeyboard(dashData)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard

	// We use Send because EditMessageText returns an EditMessageTextConfig
	// Wait, bot.Send accepts Chattable. NewEditMessageText returns EditMessageTextConfig which is Chattable.
	// But duplicate content error might occur if nothing changed, so wrap in try/catch logic (Go doesn't have try/catch, just ignore error)
	_, err = bot.Send(edit)
	if err != nil {
		// Ignore "message is not modified" error
	}
}

// IsAuthorizedCallback reports whether a callback query comes from the
// configured chat. An unset TELEGRAM_CHAT_ID rejects every callback so a
// missing .env cannot silently open the bot to forwarded messages.
func IsAuthorizedCallback(update tgbotapi.Update) bool {
	cq := update.CallbackQuery
	if cq == nil || cq.Message == nil || core.TelegramChatID == "" {
		return false
	}
	return fmt.Sprintf("%d", cq.Message.Chat.ID) == core.TelegramChatID
}

func answerUnauthorized(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if cq := update.CallbackQuery; cq != nil {
		bot.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "⛔ Không có quyền"))
	}
}
