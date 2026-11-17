package bot

import (
	"fmt"
	"super-bot/core"

	"github.com/bwmarrin/discordgo"
)

// Discord caps a message at 5 action rows (max 5 buttons per row).
// Country pages therefore show 20 countries = 4 full rows, leaving the
// 5th row for the nav buttons. The previous 25/page produced 6 rows and
// Discord rejected the whole panel (400 Invalid Form Body) — the
// interaction was already deferred, so it sat "thinking" until timeout.
// Edge pages pack 5 edges per row, 20 per page, for the same reason.
const (
	ffvpnCountriesPerPageD = 20
	ffvpnEdgesPerPageD     = 20
)

// groupCountriesByCodeD indexes edges by country code (preserving order).
func groupCountriesByCodeD(info core.FFVPNInfo) ([]string, map[string][]core.FFVPNEdge) {
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

// CreateFFVPNButtons renders country buttons (25 per page, nav row).
func CreateFFVPNButtons(info core.FFVPNInfo) []discordgo.MessageComponent {
	return ffvpnCountryButtonsD(info, 0)
}

func ffvpnCountryButtonsD(info core.FFVPNInfo, page int) []discordgo.MessageComponent {
	order, byCode := groupCountriesByCodeD(info)
	var components []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	pages := (len(order) + ffvpnCountriesPerPageD - 1) / ffvpnCountriesPerPageD
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * ffvpnCountriesPerPageD
	end := start + ffvpnCountriesPerPageD
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
		style := discordgo.SecondaryButton
		emoji := "🌐"
		if currentHere {
			style = discordgo.SuccessButton
			emoji = "🟢"
		}
		btn := discordgo.Button{
			Label:    code,
			Style:    style,
			CustomID: "ffvpn_k_" + code + "_" + fmt.Sprint(page),
			Emoji:    &discordgo.ComponentEmoji{Name: emoji},
		}
		row = append(row, btn)
		if len(row) == 5 {
			components = append(components, discordgo.ActionsRow{Components: row})
			row = []discordgo.MessageComponent{}
		}
	}
	if len(row) > 0 {
		components = append(components, discordgo.ActionsRow{Components: row})
	}

	if pages > 1 {
		var nav []discordgo.MessageComponent
		if page > 0 {
			nav = append(nav, discordgo.Button{
				Label: "◀️ Trước", Style: discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("ffvpn_p_%d", page-1),
			})
		}
		if page < pages-1 {
			nav = append(nav, discordgo.Button{
				Label: "Sau ▶️", Style: discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("ffvpn_p_%d", page+1),
			})
		}
		nav = append(nav, discordgo.Button{
			Label: "🔄 Refresh", Style: discordgo.PrimaryButton,
			CustomID: "ffvpn_refresh",
			Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
		})
		components = append(components, discordgo.ActionsRow{Components: nav})
	} else {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "🔄 Refresh",
					Style:    discordgo.PrimaryButton,
					CustomID: "ffvpn_refresh",
					Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
				},
			},
		})
	}
	return components
}

// CreateFFVPNEdgeButtonsD renders the edges of one country + back button.
func CreateFFVPNEdgeButtonsD(info core.FFVPNInfo, code string, page int) []discordgo.MessageComponent {
	_, byCode := groupCountriesByCodeD(info)
	var components []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	edges := byCode[code]
	pages := (len(edges) + ffvpnEdgesPerPageD - 1) / ffvpnEdgesPerPageD
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * ffvpnEdgesPerPageD
	end := start + ffvpnEdgesPerPageD
	if end > len(edges) {
		end = len(edges)
	}

	for _, e := range edges[start:end] {
		emoji := "🌐"
		if e.ServerHost == info.Current.ServerHost {
			emoji = "✅"
		}
		btn := discordgo.Button{
			Label:    fmt.Sprintf("%s:%d", e.CityName, e.ServerPort),
			Style:    discordgo.SecondaryButton,
			CustomID: "ffvpn_e_" + e.ServerHost,
			Emoji:    &discordgo.ComponentEmoji{Name: emoji},
		}
		row = append(row, btn)
		if len(row) == 5 {
			components = append(components, discordgo.ActionsRow{Components: row})
			row = []discordgo.MessageComponent{}
		}
	}
	if len(row) > 0 {
		components = append(components, discordgo.ActionsRow{Components: row})
	}

	var nav []discordgo.MessageComponent
	if page > 0 {
		nav = append(nav, discordgo.Button{
			Label: "◀️ Trước", Style: discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("ffvpn_ke_%s_%d", code, page-1),
		})
	}
	if page < pages-1 {
		nav = append(nav, discordgo.Button{
			Label: "Sau ▶️", Style: discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("ffvpn_ke_%s_%d", code, page+1),
		})
	}
	nav = append(nav, discordgo.Button{
		Label:    "◀️ Quốc gia",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("ffvpn_p_%d", page),
	})
	components = append(components, discordgo.ActionsRow{Components: nav})
	return components
}

// CreateFFVPNEmbed renders the ffvpn panel as an embed.
func CreateFFVPNEmbed(info core.FFVPNInfo) *discordgo.MessageEmbed {
	state := "stopped"
	if info.Status.Running {
		state = "running"
	}
	color := 0x888888
	if info.Status.Running {
		color = 0x2ecc71
	}
	description := fmt.Sprintf(
		"**Server:** %s\n**Proxy:** %s\n**Account:** %s\n**Edges:** %d",
		ffvpnDisplayD(info.Current), state, valD(info.Status.Account), len(info.Servers))
	if info.Status.EdgeSuspect {
		description += "\n⚠️ This edge is rejecting tunnels server-side — pick another server."
	}
	return &discordgo.MessageEmbed{
		Title:       "🦊 FFVPN",
		Description: description,
		Color:       color,
	}
}

func ffvpnDisplayD(e core.FFVPNEdge) string {
	return fmt.Sprintf("%s/%s (%s:%d)", e.CountryName, e.CityName, e.ServerHost, e.ServerPort)
}

func valD(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
