package telegram

import (
	"fmt"
	"strings"
	"super-bot/core"
)

// FormatDashboardMessage formats the dashboard data into a Telegram Markdown message
func FormatDashboardMessage(data *core.DashboardData) string {
	var sb strings.Builder

	// Helper to handle empty strings
	val := func(s string) string {
		if s == "" {
			return "N/A"
		}
		return s
	}

	// --- Proxmox Section ---
	if data.Proxmox.Error != "" {
		sb.WriteString("🏗 *PROXMOX:* ❌ Lỗi kết nối\n")
	} else {
		sb.WriteString(fmt.Sprintf("🏗 *PROXMOX VE:* `%s`\n", val(data.Proxmox.Node)))
		sb.WriteString(fmt.Sprintf("⏱️ Uptime: `%s`\n", val(data.Proxmox.Uptime)))

		for _, vm := range data.Proxmox.VMs {
			icon := "📦"
			if vm.Type == "qemu" {
				icon = "🖥"
			}
			status := "❌"
			if vm.Status == "running" {
				status = "✅"
			}
			sb.WriteString(fmt.Sprintf(" • %s %s: %s\n", icon, vm.Name, status))
		}
	}

	sb.WriteString("----------------------------\n")

	// --- MikroTik Section ---
	if data.MikroTik.Error != "" {
		sb.WriteString(fmt.Sprintf("📟 *MIKROTIK:* ❌ Lỗi: %s\n", data.MikroTik.Error))
	} else {
		sb.WriteString(fmt.Sprintf("📟 *MIKROTIK:* `%s`\n", val(data.MikroTik.Name)))
		sb.WriteString(fmt.Sprintf("📊 CPU: `%s%%` | RAM: `%s`\n", val(data.MikroTik.CPU), val(data.MikroTik.RAM)))
		sb.WriteString(fmt.Sprintf("⏱ Uptime: `%s`\n", val(data.MikroTik.Uptime)))
	}

	// --- PPPoE Section ---
	if data.PPPoE.Error != "" {
		sb.WriteString("🌐 PPPoE: ❌ Lỗi kết nối\n")
	} else {
		sb.WriteString(fmt.Sprintf("🌐 PPPoE: ↓ `%.2f Mbps` | ↑ `%.2f Mbps`\n", data.PPPoE.RxSpeed, data.PPPoE.TxSpeed))
	}

	sb.WriteString("----------------------------\n")

	// --- Sing-box Section ---
	if data.Singbox.Error != "" {
		sb.WriteString(fmt.Sprintf("⚡️ *Sing-box:* ❌ Lỗi: %s\n", data.Singbox.Error))
	} else {
		sb.WriteString(fmt.Sprintf("⚡️ *Đang Chọn:* `%s`\n", val(data.Singbox.CurrentNode)))
	}

	// Footer
	sb.WriteString(fmt.Sprintf("🕒 _Cập nhật lúc: %s_", data.Timestamp))

	return sb.String()
}
