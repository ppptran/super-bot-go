package core

import (
	"context"
	"fmt"
	"time"
)

// GetDashboardData aggregates all monitoring data using goroutines and channels
// This is the main optimization: all data sources are fetched concurrently
func GetDashboardData(ctx context.Context) (*DashboardData, error) {
	// Create channels for each data source
	mikrotikChan := make(chan MikroTikInfo, 1)
	proxmoxChan := make(chan ProxmoxInfo, 1)
	singboxChan := make(chan SingboxInfo, 1)
	pppoeChan := make(chan PPPoESpeed, 1)

	// Launch goroutines to fetch data concurrently
	go GetMikroTikInfo(ctx, mikrotikChan)
	go GetProxmoxInfo(ctx, proxmoxChan)
	go GetSingboxInfo(ctx, singboxChan)
	go GetPPPoESpeed(ctx, pppoeChan)

	// Create timeout context (max 5 seconds for all operations)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Collect results from channels
	var mikrotik MikroTikInfo
	var proxmox ProxmoxInfo
	var singbox SingboxInfo
	var pppoe PPPoESpeed

	// Wait for all results or timeout
	resultsReceived := 0
	for resultsReceived < 4 {
		select {
		case m, ok := <-mikrotikChan:
			if ok {
				mikrotik = m
				resultsReceived++
				mikrotikChan = nil
			}
		case p, ok := <-proxmoxChan:
			if ok {
				proxmox = p
				resultsReceived++
				proxmoxChan = nil
			}
		case s, ok := <-singboxChan:
			if ok {
				singbox = s
				resultsReceived++
				singboxChan = nil
			}
		case v, ok := <-pppoeChan:
			if ok {
				pppoe = v
				resultsReceived++
				pppoeChan = nil
			}
		case <-timeoutCtx.Done():
			// Timeout: return partial data
			return nil, fmt.Errorf("timeout fetching dashboard data")
		}
	}

	// Get current timestamp
	timestamp := GetVietnamTime()

	fmt.Printf("DEBUG: Dashboard Data:\nProxmox: %+v\nMikroTik: %+v\nPPPoE: %+v\nSingbox: %+v\n",
		proxmox, mikrotik, pppoe, singbox)

	return &DashboardData{
		MikroTik:  mikrotik,
		Proxmox:   proxmox,
		Singbox:   singbox,
		PPPoE:     pppoe,
		Timestamp: timestamp,
	}, nil
}
