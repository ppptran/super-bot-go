package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"super-bot/core"
	"time"

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

	text := FormatFFVPNMessage(info)
	keyboard := CreateFFVPNKeyboard(info)
	if info.Error != "" {
		text = "❌ ffvpn: " + info.Error
	}
	sendFFVPNPanel(bot, update.Message.Chat.ID, sent.MessageID, text, keyboard)
}

// sendFFVPNPanel edits the panel in place. "message is not modified" is a
// no-op, not a failure; any other edit error falls back to a fresh message
// so the panel never silently disappears.
func sendFFVPNPanel(bot *tgbotapi.BotAPI, chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	if _, err := bot.Send(edit); err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return
		}
		log.Printf("ffvpn: edit failed (%v), sending a fresh panel", err)
		fallback := tgbotapi.NewMessage(chatID, text)
		fallback.ParseMode = "Markdown"
		fallback.ReplyMarkup = &keyboard
		if _, ferr := bot.Send(fallback); ferr != nil {
			log.Printf("ffvpn: fallback send failed: %v", ferr)
		}
	}
}

// answerFFVPNCallback answers a callback query exactly once. Telegram rejects
// a second answer for the same query ID, which used to swallow the real
// success/error toast.
func answerFFVPNCallback(bot *tgbotapi.BotAPI, id, text string, alert bool) {
	var cfg tgbotapi.CallbackConfig
	if alert {
		cfg = tgbotapi.NewCallbackWithAlert(id, text)
	} else {
		cfg = tgbotapi.NewCallback(id, text)
	}
	if _, err := bot.Request(cfg); err != nil {
		log.Printf("ffvpn: answer callback: %v", err)
	}
}

// parseFFVPNEdgeRef parses the short "<countryCode>|<index>" edge payload.
func parseFFVPNEdgeRef(ref string) (string, int, bool) {
	parts := strings.SplitN(ref, "|", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil || idx < 0 {
		return "", 0, false
	}
	return parts[0], idx, true
}

// switchFFVPNEdgeRef resolves a short ref against the current edge list and
// switches to it. Keeping callback data to code+index avoids Telegram's
// 64-byte callback_data limit, which long hostnames can exceed.
func switchFFVPNEdgeRef(bot *tgbotapi.BotAPI, callbackID, code string, idx int) bool {
	ctx, cancel := ffvpnContext()
	defer cancel()
	ch := make(chan core.FFVPNInfo, 1)
	core.GetFFVPNInfo(ctx, ch)
	info := <-ch
	if info.Error != "" {
		answerFFVPNCallback(bot, callbackID, "❌ "+info.Error, true)
		return false
	}
	edge, ok := core.FindFFVPNEdge(info, code, idx)
	if !ok {
		answerFFVPNCallback(bot, callbackID, "❌ Edge không còn tồn tại — bấm Refresh", true)
		return false
	}
	if err := core.SwitchFFVPNEdge(edge); err != nil {
		answerFFVPNCallback(bot, callbackID, "❌ Lỗi: "+err.Error(), true)
		return false
	}
	answerFFVPNCallback(bot, callbackID, fmt.Sprintf("✅ Đã chuyển sang %s:%d", edge.CityName, edge.ServerPort), false)
	return true
}

// HandleFFVPNButtonCallback handles "ffvpn_e|<code>|<idx>" and navigation payloads.
func HandleFFVPNButtonCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if !IsAuthorizedCallback(update) {
		answerUnauthorized(bot, update)
		return
	}

	callback := update.CallbackQuery
	data := callback.Data

	// Answer once: switch actions answer after the API call so the toast
	// carries the real result; navigation answers immediately.
	switch {
	case data == "ffvpnrefresh":
		answerFFVPNCallback(bot, callback.ID, "🔄 Đang cập nhật...", false)
	case data == "ffvpn_noop", strings.HasPrefix(data, "ffvpn_c|"):
		answerFFVPNCallback(bot, callback.ID, "", false)
	case strings.HasPrefix(data, "ffvpn_e|"):
		ref := strings.TrimPrefix(data, "ffvpn_e|")
		if code, idx, ok := parseFFVPNEdgeRef(ref); ok {
			if !switchFFVPNEdgeRef(bot, callback.ID, code, idx) {
				return
			}
		} else {
			// legacy "<host>" payload kept for panels sent before short refs
			if err := core.SwitchFFVPN(ref); err != nil {
				answerFFVPNCallback(bot, callback.ID, "❌ Lỗi: "+err.Error(), true)
				return
			}
			answerFFVPNCallback(bot, callback.ID, "✅ Đã chuyển sang "+ref, false)
		}
	case strings.HasPrefix(data, "ffvpn|"):
		// legacy single-tap payload (kept for old messages)
		host := strings.TrimPrefix(data, "ffvpn|")
		if err := core.SwitchFFVPN(host); err != nil {
			answerFFVPNCallback(bot, callback.ID, "❌ Lỗi: "+err.Error(), true)
			return
		}
		answerFFVPNCallback(bot, callback.ID, "✅ Đã chuyển sang "+host, false)
	default:
		answerFFVPNCallback(bot, callback.ID, "", false)
	}

	if callback.Message == nil {
		return
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
	sendFFVPNPanel(bot, callback.Message.Chat.ID, callback.Message.MessageID, text, keyboard)
}

// ffvpnKeyboardForCallback renders the keyboard matching the callback:
// country page vs expanded country edges.
func ffvpnKeyboardForCallback(info core.FFVPNInfo, data string) tgbotapi.InlineKeyboardMarkup {
	order, byCode := groupCountriesByCode(info)

	if strings.HasPrefix(data, "ffvpn_ke|") {
		parts := strings.Split(strings.TrimPrefix(data, "ffvpn_ke|"), "|")
		if len(parts) == 3 {
			countryPage, _ := strconv.Atoi(parts[1])
			edgePage, _ := strconv.Atoi(parts[2])
			return ffvpnEdgePage(byCode, info, parts[0], edgePage, countryPage)
		}
	}
	if strings.HasPrefix(data, "ffvpn_k|") {
		parts := strings.SplitN(strings.TrimPrefix(data, "ffvpn_k|"), "|", 2)
		countryPage := 0
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &countryPage)
		}
		return ffvpnEdgePage(byCode, info, parts[0], 0, countryPage)
	}
	page := 0
	if strings.HasPrefix(data, "ffvpn_c|") {
		fmt.Sscanf(strings.TrimPrefix(data, "ffvpn_c|"), "%d", &page)
	}
	return ffvpnCountryPage(order, byCode, info, page)
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

// Callback payloads are stateless (no server-side session storage needed):
//
//	"ffvpn_c|<page>"                           — country page N
//	"ffvpn_k|<code>|<countryPage>"             — edges of <code>, page 0
//	"ffvpn_ke|<code>|<countryPage>|<edgePage>" — edge page navigation
//	"ffvpn_e|<code>|<index>"                   — switch to an edge (short ref)
//	"ffvpnrefresh"                             — re-render country page 0
//
// Edge refs carry a country code + index instead of the hostname so the
// payload stays under Telegram's 64-byte callback_data limit.
const (
	ffvpnCountriesPerPage = 20
	ffvpnEdgesPerPage     = 20
)

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

// ffvpnEdgePage renders one page of a country's edges plus pagination and a
// back button that returns to the country page the user came from.
func ffvpnEdgePage(byCode map[string][]core.FFVPNEdge, info core.FFVPNInfo, code string, page, countryPage int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	edges := byCode[code]
	pages := (len(edges) + ffvpnEdgesPerPage - 1) / ffvpnEdgesPerPage
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * ffvpnEdgesPerPage
	end := start + ffvpnEdgesPerPage
	if end > len(edges) {
		end = len(edges)
	}

	for i, e := range edges[start:end] {
		icon := "🌐"
		if e.ServerHost == info.Current.ServerHost {
			icon = "✅"
		}
		label := fmt.Sprintf("%s %s:%d", icon, e.CityName, e.ServerPort)
		ref := fmt.Sprintf("ffvpn_e|%s|%d", code, start+i)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(label, ref),
		})
	}

	if pages > 1 {
		var nav []tgbotapi.InlineKeyboardButton
		if page > 0 {
			nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(
				"◀️ Trước", fmt.Sprintf("ffvpn_ke|%s|%d|%d", code, countryPage, page-1)))
		}
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("📄 %d/%d", page+1, pages), "ffvpn_noop"))
		if page < pages-1 {
			nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(
				"Sau ▶️", fmt.Sprintf("ffvpn_ke|%s|%d|%d", code, countryPage, page+1)))
		}
		rows = append(rows, nav)
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("◀️ Quốc gia", fmt.Sprintf("ffvpn_c|%d", countryPage)),
	})
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
