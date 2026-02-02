package core

import "time"

// DashboardData aggregates all monitoring data
type DashboardData struct {
	MikroTik  MikroTikInfo
	Proxmox   ProxmoxInfo
	Singbox   SingboxInfo
	VNPT      VNPTSpeed
	Timestamp string
}

// MikroTikInfo contains MikroTik router information
type MikroTikInfo struct {
	Name   string
	CPU    string
	RAM    string
	Uptime string
	Error  string
}

// ProxmoxInfo contains Proxmox VE information
type ProxmoxInfo struct {
	Node   string
	Uptime string
	VMs    []VMInfo
	Error  string
}

// VMInfo contains VM/container information
type VMInfo struct {
	Name   string
	Type   string // "qemu" or "lxc"
	Status string // "running" or "stopped"
}

// SingboxInfo contains Sing-box VPN information
type SingboxInfo struct {
	CurrentNode string
	AllNodes    []string
	NodeDelays  map[string]int
	Error       string
}

// VNPTSpeed contains VNPT bandwidth information
type VNPTSpeed struct {
	RxSpeed float64 // Download speed in Mbps
	TxSpeed float64 // Upload speed in Mbps
	Error   string
}

// GetVietnamTime returns current time in Vietnam timezone (UTC+7)
func GetVietnamTime() string {
	loc := time.FixedZone("UTC+7", 7*60*60)
	return time.Now().In(loc).Format("15:04:05")
}
