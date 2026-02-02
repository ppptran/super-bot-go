package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// GetProxmoxInfo fetches Proxmox VE information via API using goroutine
func GetProxmoxInfo(ctx context.Context, resultChan chan<- ProxmoxInfo) {
	defer close(resultChan)

	// Create HTTP client with timeout and skip SSL verification
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Build API URL
	baseURL := fmt.Sprintf("https://%s:8006/api2/json", PVEHost)
	authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", PVEUser, PVETokenName, PVETokenValue)

	// Get nodes
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/nodes", nil)
	if err != nil {
		resultChan <- ProxmoxInfo{Error: err.Error()}
		return
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		resultChan <- ProxmoxInfo{Error: err.Error()}
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var nodesResp struct {
		Data []struct {
			Node   string `json:"node"`
			Uptime int    `json:"uptime"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &nodesResp); err != nil || len(nodesResp.Data) == 0 {
		resultChan <- ProxmoxInfo{Error: "Failed to parse nodes"}
		return
	}

	nodeName := nodesResp.Data[0].Node
	uptime := nodesResp.Data[0].Uptime
	uptimeStr := fmt.Sprintf("%dd %02dh", uptime/86400, (uptime%86400)/3600)

	// Get VMs and containers
	req2, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/cluster/resources?type=vm", nil)
	if err != nil {
		resultChan <- ProxmoxInfo{Node: nodeName, Uptime: uptimeStr, Error: err.Error()}
		return
	}
	req2.Header.Set("Authorization", authHeader)

	resp2, err := client.Do(req2)
	if err != nil {
		resultChan <- ProxmoxInfo{Node: nodeName, Uptime: uptimeStr, Error: err.Error()}
		return
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	var vmsResp struct {
		Data []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body2, &vmsResp); err != nil {
		resultChan <- ProxmoxInfo{Node: nodeName, Uptime: uptimeStr, Error: err.Error()}
		return
	}

	// Convert to VMInfo
	vms := make([]VMInfo, 0, len(vmsResp.Data))
	for _, vm := range vmsResp.Data {
		vms = append(vms, VMInfo{
			Name:   vm.Name,
			Type:   vm.Type,
			Status: vm.Status,
		})
	}

	// Sort by name
	sort.Slice(vms, func(i, j int) bool {
		return vms[i].Name < vms[j].Name
	})

	resultChan <- ProxmoxInfo{
		Node:   nodeName,
		Uptime: uptimeStr,
		VMs:    vms,
	}
}
