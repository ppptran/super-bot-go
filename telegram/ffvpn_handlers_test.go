package telegram

import (
	"fmt"
	"strings"
	"testing"

	"super-bot/core"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func makeFFVPNInfo(countries int) core.FFVPNInfo {
	servers := make([]core.FFVPNEdge, 0, countries)
	for i := 0; i < countries; i++ {
		servers = append(servers, core.FFVPNEdge{
			CountryCode: fmt.Sprintf("C%02d", i),
			CountryName: fmt.Sprintf("Country %02d", i),
			CityName:    "City",
			ServerHost:  fmt.Sprintf("host%02d.example", i),
			ServerPort:  443,
		})
	}
	return core.FFVPNInfo{Servers: servers}
}

func TestFFVPNCountryPagesDoNotOverlap(t *testing.T) {
	info := makeFFVPNInfo(33)
	order, byCode := groupCountriesByCode(info)

	page0 := ffvpnCountryPage(order, byCode, info, 0)
	page1 := ffvpnCountryPage(order, byCode, info, 1)

	if got, want := len(page0.InlineKeyboard), 7; got != want {
		t.Fatalf("page 0 rows = %d, want %d", got, want)
	}
	if got, want := len(page1.InlineKeyboard), 6; got != want {
		t.Fatalf("page 1 rows = %d, want %d", got, want)
	}

	seen := map[string]bool{}
	for _, kb := range []tgbotapi.InlineKeyboardMarkup{page0, page1} {
		for _, row := range kb.InlineKeyboard {
			for _, btn := range row {
				if btn.CallbackData == nil || !strings.HasPrefix(*btn.CallbackData, "ffvpn_k|") {
					continue
				}
				if seen[*btn.CallbackData] {
					t.Fatalf("duplicate country button %q", *btn.CallbackData)
				}
				seen[*btn.CallbackData] = true
			}
		}
	}
	if got := len(seen); got != 33 {
		t.Fatalf("countries rendered = %d, want 33", got)
	}
}

func TestFFVPNEdgePageBackUsesCountryPage(t *testing.T) {
	info := makeFFVPNInfo(33)
	order, byCode := groupCountriesByCode(info)

	kb := ffvpnEdgePage(order, byCode, info, "C00", 1)
	rows := kb.InlineKeyboard
	back := rows[len(rows)-1][0]
	if back.CallbackData == nil || *back.CallbackData != "ffvpn_c|1" {
		t.Fatalf("back callback = %v, want ffvpn_c|1", back.CallbackData)
	}
}

func TestIsAuthorizedCallback(t *testing.T) {
	old := core.TelegramChatID
	t.Cleanup(func() { core.TelegramChatID = old })

	cq := &tgbotapi.CallbackQuery{
		ID:      "1",
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 123}},
	}

	core.TelegramChatID = "123"
	if !IsAuthorizedCallback(tgbotapi.Update{CallbackQuery: cq}) {
		t.Fatal("callback from the configured chat was rejected")
	}

	cq.Message.Chat.ID = 456
	if IsAuthorizedCallback(tgbotapi.Update{CallbackQuery: cq}) {
		t.Fatal("callback from another chat was accepted")
	}

	core.TelegramChatID = ""
	if IsAuthorizedCallback(tgbotapi.Update{CallbackQuery: cq}) {
		t.Fatal("callback was accepted with an empty TELEGRAM_CHAT_ID")
	}
}
