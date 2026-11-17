package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"super-bot/core"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ffvpnContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

// HandleFFVPNStatusCommand handles /ffvpn — current server + proxy state.
func HandleFFVPNStatusCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🔄 *Đang tải ffvpn...*")
	msg.ParseMode = "Markdown"
	sent, err := bot.Send(msg)
	if err != nil {
		log.Println("ffvpn: send:", err)
		return
	}

	ctx, cancel := ffvpnContext()
	defer cancel()
	ch := make(chan core.FFVPNInfo, 1)
	core.GetFFVPNInfo(ctx, ch)
	info := <-ch

	// Render, preferring an in-place edit of the "Đang tải..." message;
	// any edit failure (stale message, rate limit, not-modified) falls back
	// to sending a fresh panel message so the panel ALWAYS appears — a
	// swallowed failure is what made the first /ffvpn look ignored.
	text := FormatFFVPNMessage(info)
	keyboard := CreateFFVPNKeyboard(info)
	if info.Error != "" {
		text = "❌ ffvpn: " + info.Error
	}
	edit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, sent.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	if _, err := bot.Send(edit); err != nil {
		fallback := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		fallback.ParseMode = "Markdown"
		fallback.ReplyMarkup = &keyboard
		bot.Send(fallback)
	}
}

// HandleFFVPNButtonCallback handles "ffvpn|<host>" and "ffvpnrefresh".
func HandleFFVPNButtonCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	callback := update.CallbackQuery
	data := callback.Data

	switch {
	case data == "ffvpnrefresh":
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Đang cập nhật..."))
	case data == "ffvpn_noop":
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case strings.HasPrefix(data, "ffvpn_c|"):
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	case strings.HasPrefix(data, "ffvpn_e|"):
		host := strings.TrimPrefix(data, "ffvpn_e|")
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Đang chuyển sang "+host+"..."))
		if err := core.SwitchFFVPN(host); err != nil {
			bot.Request(tgbotapi.NewCallbackWithAlert(callback.ID, "❌ Lỗi: "+err.Error()))
		} else {
			bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Đã chuyển sang "+host))
		}
	case strings.HasPrefix(data, "ffvpn|"):
		// legacy single-tap payload (kept for old messages)
		host := strings.TrimPrefix(data, "ffvpn|")
		bot.Request(tgbotapi.NewCallback(callback.ID, "🔄 Đang chuyển sang "+host+"..."))
		if err := core.SwitchFFVPN(host); err != nil {
			bot.Request(tgbotapi.NewCallbackWithAlert(callback.ID, "❌ Lỗi: "+err.Error()))
		} else {
			bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Đã chuyển sang "+host))
		}
	}

	ctx, cancel := ffvpnContext()
	defer cancel()
	ch := make(chan core.FFVPNInfo, 1)
	core.GetFFVPNInfo(ctx, ch)
	info := <-ch
	if info.Error != "" {
		return
	}
	text := FormatFFVPNMessage(info)
	keyboard := ffvpnKeyboardForCallback(info, data)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	if _, err := bot.Send(edit); err != nil {
		// Edited message may be gone/unmodified — send a fresh panel so the
		// interaction never silently dies.
		fallback := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
		fallback.ParseMode = "Markdown"
		fallback.ReplyMarkup = &keyboard
		bot.Send(fallback)
	}
}

// ffvpnKeyboardForCallback renders the keyboard matching the callback:
// country page vs expanded country edges.
func ffvpnKeyboardForCallback(info core.FFVPNInfo, data string) tgbotapi.InlineKeyboardMarkup {
	order, byCode := groupCountriesByCode(info)
	switch {
	case strings.HasPrefix(data, "ffvpn_k|"):
		parts := strings.SplitN(strings.TrimPrefix(data, "ffvpn_k|"), "|", 2)
		code := parts[0]
		page := 0
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &page)
		}
		return ffvpnEdgePage(order, byCode, info, code, page)
	case strings.HasPrefix(data, "ffvpn_c|"):
		page := 0
		fmt.Sscanf(strings.TrimPrefix(data, "ffvpn_c|"), "%d", &page)
		return ffvpnCountryPage(order, byCode, info, page)
	default:
		return ffvpnCountryPage(order, byCode, info, 0)
	}
}

// FormatFFVPNMessage renders the ffvpn panel.
func FormatFFVPNMessage(info core.FFVPNInfo) string {
	var sb strings.Builder
	sb.WriteString("🦊 *FFVPN*\n")
	sb.WriteString(fmt.Sprintf("📍 Server: `%s`\n", ffvpnDisplay(info.Current)))
	sb.WriteString(fmt.Sprintf("🔗 Proxy: *%s* (account `%s`)\n",
		onOffT(info.Status.Running), valT(info.Status.Account)))
	if info.Status.EdgeSuspect {
		sb.WriteString("⚠️ _Edge này đang từ chối tunnel — chọn server khác._\n")
	}
	sb.WriteString(fmt.Sprintf("🌐 %d edges khả dụng\n", len(info.Servers)))
	sb.WriteString("Bấm một server để đổi — sing-box không cần đổi gì.")
	return sb.String()
}

// ffvpnPage and ffvpnSel carry keyboard state through callback payloads —
// stateless (no server-side session storage needed).
//
//	"ffvpn_c|<page>"        — country page N (20 countries per page)
//	"ffvpn_k|<code>|<page>" — edges of country <code>, from page <page>
//	"ffvpn_e|<host>"        — switch to edge <host>
//	"ffvpnrefresh"          — re-render country page 0

const ffvpnCountriesPerPage = 20

// groupCountriesByCode indexes edges by country code (preserving order).
func groupCountriesByCode(info core.FFVPNInfo) ([]string, map[string][]core.FFVPNEdge) {
	var order []string
	byCode := map[string][]core.FFVPNEdge{}
	seen := map[string]bool{}
	for _, e := range info.Servers {
		if !seen[e.CountryCode] {
			seen[e.CountryCode] = true
			order = append(order, e.CountryCode)
		}
		byCode[e.CountryCode] = append(byCode[e.CountryCode], e)
	}
	return order, byCode
}

// countryName resolves a display name for a code from the edge list.
func countryName(info core.FFVPNInfo, code string) string {
	for _, e := range info.Servers {
		if e.CountryCode == code {
			return e.CountryName
		}
	}
	return code
}

// CreateFFVPNKeyboard renders the country page (or a country's edges when
// selected). Current country marked 🟢, current edge marked ✅.
func CreateFFVPNKeyboard(info core.FFVPNInfo) tgbotapi.InlineKeyboardMarkup {
	order, byCode := groupCountriesByCode(info)
	return ffvpnCountryPage(order, byCode, info, 0)
}

// ffvpnCountryPage renders one page of country buttons.
func ffvpnCountryPage(order []string, byCode map[string][]core.FFVPNEdge, info core.FFVPNInfo, page int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	pages := (len(order) + ffvpnCountriesPerPage - 1) / ffvpnCountriesPerPage
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * ffvpnCountriesPerPage
	end := start + ffvpnCountriesPerPage
	if end > len(order) {
		end = len(order)
	}

	for _, code := range order[start:end] {
		edges := byCode[code]
		currentHere := false
		for _, e := range edges {
			if e.ServerHost == info.Current.ServerHost {
				currentHere = true
			}
		}
		icon := "🌐"
		if currentHere {
			icon = "🟢"
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(icon+" "+code, "ffvpn_k|"+code+"|"+fmt.Sprint(page))
		row = append(row, btn)
		if len(row) == 4 {
			rows = append(rows, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	// Pagination row.
	if pages > 1 {
		var nav []tgbotapi.InlineKeyboardButton
		if page > 0 {
			nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("◀️ Trước", fmt.Sprintf("ffvpn_c|%d", page-1)))
		}
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📄 %d/%d", page+1, pages), "ffvpn_noop"))
		if page < pages-1 {
			nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Sau ▶️", fmt.Sprintf("ffvpn_c|%d", page+1)))
		}
		rows = append(rows, nav)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "ffvpnrefresh"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ffvpnEdgePage renders the edges of one country + a back button.
func ffvpnEdgePage(order []string, byCode map[string][]core.FFVPNEdge, info core.FFVPNInfo, code string, page int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, e := range byCode[code] {
		icon := "🌐"
		if e.ServerHost == info.Current.ServerHost {
			icon = "✅"
		}
		label := fmt.Sprintf("%s %s:%d", icon, e.CityName, e.ServerPort)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(label, "ffvpn_e|"+e.ServerHost),
		})
	}
	var nav []tgbotapi.InlineKeyboardButton
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("◀️ Quốc gia", fmt.Sprintf("ffvpn_c|%d", page)))
	rows = append(rows, nav)
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func ffvpnDisplay(e core.FFVPNEdge) string {
	return fmt.Sprintf("%s/%s (%s:%d)", e.CountryName, e.CityName, e.ServerHost, e.ServerPort)
}

func onOffT(b bool) string {
	if b {
		return "running"
	}
	return "stopped"
}

func valT(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
