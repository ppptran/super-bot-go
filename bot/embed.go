package bot

import (
	"fmt"
	"super-bot/core"

	"github.com/bwmarrin/discordgo"
)

// CreateDashboardEmbed creates a Discord embed from dashboard data
func CreateDashboardEmbed(data *core.DashboardData) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Color: 0x3498db, // Blue color
	}

	// Proxmox section
	pveValue := ""
	if data.Proxmox.Error != "" {
		pveValue = "❌ Lỗi kết nối"
	} else {
		vmLines := ""
		for _, vm := range data.Proxmox.VMs {
			icon := "🖥️"
			if vm.Type == "lxc" {
				icon = "📦"
			}
			status := "✅"
			if vm.Status != "running" {
				status = "❌"
			}
			vmLines += fmt.Sprintf("%s %s: %s\n", icon, vm.Name, status)
		}

		pveValue = fmt.Sprintf("**Node:** `%s`\n**Uptime:** `%s`\n%s",
			data.Proxmox.Node, data.Proxmox.Uptime, vmLines)
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🏗️ PROXMOX VE",
		Value:  pveValue,
		Inline: false,
	})

	// MikroTik section
	mtValue := ""
	if data.MikroTik.Error != "" {
		mtValue = fmt.Sprintf("❌ Lỗi: %s", data.MikroTik.Error)
	} else {
		mtValue = fmt.Sprintf("**Router:** `%s`\n**CPU:** `%s%%` | **RAM:** `%s`\n**Uptime:** `%s`\n**VNPT:** ↓ `%.2f Mbps` | ↑ `%.2f Mbps`",
			data.MikroTik.Name,
			data.MikroTik.CPU,
			data.MikroTik.RAM,
			data.MikroTik.Uptime,
			data.VNPT.RxSpeed,
			data.VNPT.TxSpeed,
		)
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "📟 MIKROTIK",
		Value:  mtValue,
		Inline: false,
	})

	// Sing-box section
	sbValue := ""
	if data.Singbox.Error != "" {
		sbValue = fmt.Sprintf("❌ Lỗi: %s", data.Singbox.Error)
	} else {
		sbValue = fmt.Sprintf("**Đang chọn:** `%s`\n🕒 Cập nhật lúc: `%s`",
			data.Singbox.CurrentNode,
			data.Timestamp,
		)
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "⚡ VPN EXIT NODE",
		Value:  sbValue,
		Inline: false,
	})

	return embed
}
