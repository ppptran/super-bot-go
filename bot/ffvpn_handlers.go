package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"super-bot/core"

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
	customID := i.MessageComponentData().CustomID
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	// Payload routing:
	//   ffvpn_e_<host>       — switch edge
	//   ffvpn_k_<code>_<page> — expand country edges
	//   ffvpn_ke_<code>_<page> — edge page navigation inside a country
	//   ffvpn_p_<page>       — country page navigation
	//   ffvpn_refresh        — re-render page 0
	switch {
	case strings.HasPrefix(customID, "ffvpn_e_"):
		host := strings.TrimPrefix(customID, "ffvpn_e_")
		if err := core.SwitchFFVPN(host); err != nil {
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "❌ " + err.Error(),
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("✅ Đã chuyển sang %s", host),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
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
			page := 0
			fmt.Sscanf(parts[1], "%d", &page)
			components = CreateFFVPNEdgeButtonsD(info, parts[0], page)
		}
	case strings.HasPrefix(customID, "ffvpn_ke_"):
		// Edge-page nav payload carries the country code.
		rest := strings.TrimPrefix(customID, "ffvpn_ke_")
		if idx := strings.LastIndexByte(rest, '_'); idx > 0 {
			code, pageStr := rest[:idx], rest[idx+1:]
			page := 0
			fmt.Sscanf(pageStr, "%d", &page)
			components = CreateFFVPNEdgeButtonsD(info, code, page)
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
