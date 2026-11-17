package bot

import (
	"fmt"
	"testing"

	"super-bot/core"

	"github.com/bwmarrin/discordgo"
)

func makeFFVPNCountries(n int) core.FFVPNInfo {
	countries := make([]core.FFVPNEdge, 0, n)
	for i := 0; i < n; i++ {
		countries = append(countries, core.FFVPNEdge{
			CountryCode: fmt.Sprintf("C%02d", i),
			CountryName: fmt.Sprintf("Country %02d", i),
			CityName:    "City",
			ServerHost:  fmt.Sprintf("host%02d.example", i),
			ServerPort:  443,
		})
	}
	return core.FFVPNInfo{Servers: countries}
}

func makeFFVPNCountryEdges(code string, n int) core.FFVPNInfo {
	edges := make([]core.FFVPNEdge, 0, n)
	for i := 0; i < n; i++ {
		edges = append(edges, core.FFVPNEdge{
			CountryCode: code,
			CountryName: "Country " + code,
			CityName:    fmt.Sprintf("City %02d", i),
			ServerHost:  fmt.Sprintf("%s%02d.example", code, i),
			ServerPort:  443 + i,
		})
	}
	return core.FFVPNInfo{Servers: edges}
}

// ffvpnButtons flattens the buttons of a component tree in render order.
func ffvpnButtons(components []discordgo.MessageComponent) []discordgo.Button {
	var out []discordgo.Button
	for _, c := range components {
		row, ok := c.(discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, rc := range row.Components {
			if btn, ok := rc.(discordgo.Button); ok {
				out = append(out, btn)
			}
		}
	}
	return out
}

func hasCustomID(buttons []discordgo.Button, id string) bool {
	for _, b := range buttons {
		if b.CustomID == id {
			return true
		}
	}
	return false
}

func TestFFVPNButtonsRespectDiscordRowLimit(t *testing.T) {
	if got := len(CreateFFVPNButtons(makeFFVPNCountries(33))); got > 5 {
		t.Fatalf("country page has %d action rows, Discord allows 5", got)
	}
	if got := len(ffvpnCountryButtonsD(makeFFVPNCountries(33), 1)); got > 5 {
		t.Fatalf("country page 1 has %d action rows, Discord allows 5", got)
	}

	if got := len(CreateFFVPNEdgeButtonsD(makeFFVPNCountryEdges("US", 20), "US", 0, 0)); got > 5 {
		t.Fatalf("edge page has %d action rows, Discord allows 5", got)
	}
	if got := len(CreateFFVPNEdgeButtonsD(makeFFVPNCountryEdges("US", 25), "US", 1, 0)); got > 5 {
		t.Fatalf("edge page 1 has %d action rows, Discord allows 5", got)
	}
}

func TestFFVPNEdgeButtonsUseShortRefsAndRememberCountryPage(t *testing.T) {
	info := makeFFVPNCountryEdges("US", 25)

	page0 := ffvpnButtons(CreateFFVPNEdgeButtonsD(info, "US", 0, 2))
	page1 := ffvpnButtons(CreateFFVPNEdgeButtonsD(info, "US", 1, 2))

	if len(page0) == 0 || page0[0].CustomID != "ffvpn_e_US_0" {
		t.Fatalf("first edge button = %v, want ffvpn_e_US_0", page0)
	}
	if !hasCustomID(page0, "ffvpn_e_US_19") {
		t.Fatal("page 0 is missing the 20th edge ref ffvpn_e_US_19")
	}
	if !hasCustomID(page1, "ffvpn_e_US_20") {
		t.Fatal("page 1 is missing the 21st edge ref ffvpn_e_US_20")
	}

	// Back returns to the originating country page, not page 0.
	if !hasCustomID(page0, "ffvpn_p_2") || !hasCustomID(page1, "ffvpn_p_2") {
		t.Fatal("back button lost the country page")
	}
	// Edge-page nav carries country page and edge page together.
	if !hasCustomID(page0, "ffvpn_ke_US_2_1") {
		t.Fatal("page 0 is missing next nav ffvpn_ke_US_2_1")
	}
	if !hasCustomID(page1, "ffvpn_ke_US_2_0") {
		t.Fatal("page 1 is missing prev nav ffvpn_ke_US_2_0")
	}
}

func TestFFVPNCountryButtonsCarryPage(t *testing.T) {
	buttons := ffvpnButtons(ffvpnCountryButtonsD(makeFFVPNCountries(33), 1))
	if !hasCustomID(buttons, "ffvpn_k_C20_1") {
		t.Fatalf("country button on page 1 should carry its page, got %v", buttons)
	}
	if !hasCustomID(buttons, "ffvpn_p_0") {
		t.Fatal("page 1 is missing prev nav ffvpn_p_0")
	}
	if !hasCustomID(buttons, "ffvpn_refresh") {
		t.Fatal("page 1 is missing the refresh button")
	}
}
