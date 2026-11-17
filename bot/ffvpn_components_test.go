package bot

import (
	"fmt"
	"testing"

	"super-bot/core"
)

func TestFFVPNButtonsRespectDiscordRowLimit(t *testing.T) {
	countries := make([]core.FFVPNEdge, 0, 33)
	for i := 0; i < 33; i++ {
		countries = append(countries, core.FFVPNEdge{
			CountryCode: fmt.Sprintf("C%02d", i),
			CountryName: fmt.Sprintf("Country %02d", i),
			CityName:    "City",
			ServerHost:  fmt.Sprintf("host%02d.example", i),
			ServerPort:  443,
		})
	}
	if got := len(CreateFFVPNButtons(core.FFVPNInfo{Servers: countries})); got > 5 {
		t.Fatalf("country page has %d action rows, Discord allows 5", got)
	}

	edges := make([]core.FFVPNEdge, 0, 20)
	for i := 0; i < 20; i++ {
		edges = append(edges, core.FFVPNEdge{
			CountryCode: "US",
			CountryName: "United States",
			CityName:    fmt.Sprintf("City %02d", i),
			ServerHost:  fmt.Sprintf("us%02d.example", i),
			ServerPort:  443,
		})
	}
	if got := len(CreateFFVPNEdgeButtonsD(core.FFVPNInfo{Servers: edges}, "US", 0)); got > 5 {
		t.Fatalf("edge page has %d action rows, Discord allows 5", got)
	}
}
