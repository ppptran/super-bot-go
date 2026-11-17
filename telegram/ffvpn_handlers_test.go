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

// makeFFVPNCountryInfo returns n edges that all belong to one country.
func makeFFVPNCountryInfo(code string, n int) core.FFVPNInfo {
	servers := make([]core.FFVPNEdge, 0, n)
	for i := 0; i < n; i++ {
		servers = append(servers, core.FFVPNEdge{
			CountryCode: code,
			CountryName: "Country " + code,
			CityName:    fmt.Sprintf("City %02d", i),
			ServerHost:  fmt.Sprintf("%s%02d.example", strings.ToLower(code), i),
			ServerPort:  443 + i,
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
	_, byCode := groupCountriesByCode(info)

	kb := ffvpnEdgePage(byCode, info, "C00", 0, 1)
	rows := kb.InlineKeyboard
	back := rows[len(rows)-1][0]
	if back.CallbackData == nil || *back.CallbackData != "ffvpn_c|1" {
		t.Fatalf("back callback = %v, want ffvpn_c|1", back.CallbackData)
	}
}

func TestFFVPNEdgePagination(t *testing.T) {
	info := makeFFVPNCountryInfo("US", 25)
	_, byCode := groupCountriesByCode(info)

	page0 := ffvpnEdgePage(byCode, info, "US", 0, 2)
	page1 := ffvpnEdgePage(byCode, info, "US", 1, 2)

	// 20 edges on page 0 (one button per row) + nav row + back row.
	if got, want := len(page0.InlineKeyboard), ffvpnEdgesPerPage+2; got != want {
		t.Fatalf("page 0 rows = %d, want %d", got, want)
	}
	// 5 edges on page 1 + nav row + back row.
	if got, want := len(page1.InlineKeyboard), 5+2; got != want {
		t.Fatalf("page 1 rows = %d, want %d", got, want)
	}

	// Edge refs must be short "<code>|<index>" payloads, not hostnames.
	wantRefs := []string{"ffvpn_e|US|0", "ffvpn_e|US|19"}
	got0 := page0.InlineKeyboard[0][0].CallbackData
	got19 := page0.InlineKeyboard[19][0].CallbackData
	if got0 == nil || *got0 != wantRefs[0] {
		t.Fatalf("first edge ref = %v, want %s", got0, wantRefs[0])
	}
	if got19 == nil || *got19 != wantRefs[1] {
		t.Fatalf("last edge ref on page 0 = %v, want %s", got19, wantRefs[1])
	}
	if got := page1.InlineKeyboard[0][0].CallbackData; got == nil || *got != "ffvpn_e|US|20" {
		t.Fatalf("first edge ref on page 1 = %v, want ffvpn_e|US|20", got)
	}

	// Navigation keeps both pages: country page 2, edge page 0 <-> 1.
	nav0 := page0.InlineKeyboard[ffvpnEdgesPerPage]
	if got := nav0[len(nav0)-1].CallbackData; got == nil || *got != "ffvpn_ke|US|2|1" {
		t.Fatalf("page 0 next = %v, want ffvpn_ke|US|2|1", got)
	}
	nav1 := page1.InlineKeyboard[5]
	if got := nav1[0].CallbackData; got == nil || *got != "ffvpn_ke|US|2|0" {
		t.Fatalf("page 1 prev = %v, want ffvpn_ke|US|2|0", got)
	}
}

func TestFFVPNCallbackDataUnderTelegramLimit(t *testing.T) {
	// Long hostnames used to overflow callback_data; refs must stay short
	// even with a country that has many edges.
	info := makeFFVPNCountryInfo("US", 25)
	order, byCode := groupCountriesByCode(info)

	callbacks := make([]string, 0, 64)
	add := func(kb tgbotapi.InlineKeyboardMarkup) {
		for _, row := range kb.InlineKeyboard {
			for _, btn := range row {
				if btn.CallbackData != nil {
					callbacks = append(callbacks, *btn.CallbackData)
				}
			}
		}
	}
	add(ffvpnCountryPage(order, byCode, info, 0))
	add(ffvpnEdgePage(byCode, info, "US", 0, 0))
	add(ffvpnEdgePage(byCode, info, "US", 1, 0))

	for _, data := range callbacks {
		if got := len([]byte(data)); got > 64 {
			t.Fatalf("callback_data %q is %d bytes, Telegram allows 64", data, got)
		}
	}
}

func TestParseFFVPNEdgeRef(t *testing.T) {
	code, idx, ok := parseFFVPNEdgeRef("US|3")
	if !ok || code != "US" || idx != 3 {
		t.Fatalf("parseFFVPNEdgeRef(US|3) = %q, %d, %v", code, idx, ok)
	}
	for _, bad := range []string{"", "US", "US|", "US|x", "US|-1"} {
		if _, _, ok := parseFFVPNEdgeRef(bad); ok {
			t.Fatalf("parseFFVPNEdgeRef(%q) accepted a bad ref", bad)
		}
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
