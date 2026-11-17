package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"super-bot/core"
	"time"

	"github.com/bwmarrin/discordgo"
)

func ffvpnContextD() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

// HandleFFVPNStatusCommand handles the /ffvpn slash command.
func HandleFFVPNStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	ctx, cancel := ffvpnContextD()
	defer cancel()
	ch := make(chan core.FFVPNInfo, 1)
	core.GetFFVPNInfo(ctx, ch)
	info := <-ch
	if info.Error != "" {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ ffvpn: " + info.Error),
		})
		return
	}

	embed := CreateFFVPNEmbed(info)
	components := CreateFFVPNButtons(info)
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		log.Printf("ffvpn: initial edit failed: %v", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
	}
}

// HandleFFVPNButtonClick handles "ffvpn_<host>" and "ffvpn_refresh".
func HandleFFVPNButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !authorizedChannel(i.ChannelID) {
		respondUnauthorized(s, i)
		return
	}

	customID := i.MessageComponentData().CustomID
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	// Payload routing:
	//   ffvpn_e_<code>_<idx>                     — switch edge
	//   ffvpn_k_<code>_<countryPage>             — expand country edges
	//   ffvpn_ke_<code>_<countryPage>_<edgePage> — edge page navigation
	//   ffvpn_p_<page>                           — country page navigation
	//   ffvpn_refresh                            — re-render page 0
	switch {
	case strings.HasPrefix(customID, "ffvpn_e_"):
		rest := strings.TrimPrefix(customID, "ffvpn_e_")
		if code, idx, ok := parseFFVPNEdgeRefD(rest); ok {
			if !switchFFVPNEdgeRefD(s, i, code, idx) {
				return
			}
		} else {
			// legacy "<host>" payload kept for panels sent before short refs
			if err := core.SwitchFFVPN(rest); err != nil {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "❌ " + err.Error(),
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("✅ Đã chuyển sang %s", rest),
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
	case strings.HasPrefix(customID, "ffvpn_k_") || strings.HasPrefix(customID, "ffvpn_ke_") || strings.HasPrefix(customID, "ffvpn_p_"):
		// navigation only — no switch, just re-render below
	case customID == "ffvpn_refresh":
		// re-render below
	}

	ctx, cancel := ffvpnContextD()
	defer cancel()
	ch := make(chan core.FFVPNInfo, 1)
	core.GetFFVPNInfo(ctx, ch)
	info := <-ch
	if info.Error != "" {
		// Interaction is already deferred — editing with the error beats
		// leaving the message stuck on "thinking" until timeout.
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("❌ ffvpn: " + info.Error),
		})
		return
	}

	// Render the right view for this interaction.
	components := CreateFFVPNButtons(info)
	embed := CreateFFVPNEmbed(info)
	switch {
	case strings.HasPrefix(customID, "ffvpn_k_"):
		rest := strings.TrimPrefix(customID, "ffvpn_k_")
		parts := strings.SplitN(rest, "_", 2)
		if len(parts) == 2 {
			countryPage := 0
			fmt.Sscanf(parts[1], "%d", &countryPage)
			components = CreateFFVPNEdgeButtonsD(info, parts[0], 0, countryPage)
		}
	case strings.HasPrefix(customID, "ffvpn_ke_"):
		// <code>_<countryPage>_<edgePage>; fall back to the old
		// <code>_<edgePage> shape for panels sent before this change.
		rest := strings.TrimPrefix(customID, "ffvpn_ke_")
		parts := strings.SplitN(rest, "_", 3)
		if len(parts) == 3 {
			countryPage, _ := strconv.Atoi(parts[1])
			edgePage, _ := strconv.Atoi(parts[2])
			components = CreateFFVPNEdgeButtonsD(info, parts[0], edgePage, countryPage)
		} else if len(parts) == 2 {
			edgePage, _ := strconv.Atoi(parts[1])
			components = CreateFFVPNEdgeButtonsD(info, parts[0], edgePage, 0)
		}
	case strings.HasPrefix(customID, "ffvpn_p_"):
		page := 0
		fmt.Sscanf(strings.TrimPrefix(customID, "ffvpn_p_"), "%d", &page)
		components = ffvpnCountryButtonsD(info, page)
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		log.Printf("ffvpn: interaction edit failed: %v", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
	}
}

// parseFFVPNEdgeRefD parses the short "<countryCode>_<index>" payload.
func parseFFVPNEdgeRefD(ref string) (string, int, bool) {
	sep := strings.LastIndexByte(ref, '_')
	if sep <= 0 {
		return "", 0, false
	}
	idx, err := strconv.Atoi(ref[sep+1:])
	if err != nil || idx < 0 {
		return "", 0, false
	}
	return ref[:sep], idx, true
}

// switchFFVPNEdgeRefD resolves and switches an edge referenced by country +
// index. This keeps hostnames out of CustomIDs and makes the ref port-aware.
func switchFFVPNEdgeRefD(s *discordgo.Session, i *discordgo.InteractionCreate, code string, idx int) bool {
	ctx, cancel := ffvpnContextD()
	defer cancel()
	ch := make(chan core.FFVPNInfo, 1)
	core.GetFFVPNInfo(ctx, ch)
	info := <-ch
	if info.Error != "" {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ " + info.Error,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return false
	}
	edge, ok := core.FindFFVPNEdge(info, code, idx)
	if !ok {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Edge không còn tồn tại — bấm Refresh",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return false
	}
	if err := core.SwitchFFVPNEdge(edge); err != nil {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return false
	}
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("✅ Đã chuyển sang %s:%d", edge.CityName, edge.ServerPort),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	return true
}
