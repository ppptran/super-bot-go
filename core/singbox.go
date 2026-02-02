package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GetSingboxInfo fetches Sing-box VPN information via API using goroutine
func GetSingboxInfo(ctx context.Context, resultChan chan<- SingboxInfo) {
	defer close(resultChan)

	client := &http.Client{Timeout: 3 * time.Second}

	// Get proxies
	req, err := http.NewRequestWithContext(ctx, "GET", SingboxAPI+"/proxies", nil)
	if err != nil {
		resultChan <- SingboxInfo{Error: err.Error()}
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		resultChan <- SingboxInfo{Error: err.Error()}
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var proxiesResp struct {
		Proxies map[string]struct {
			Now string   `json:"now"`
			All []string `json:"all"`
		} `json:"proxies"`
	}

	if err := json.Unmarshal(body, &proxiesResp); err != nil {
		resultChan <- SingboxInfo{Error: err.Error()}
		return
	}

	exitNode, ok := proxiesResp.Proxies["ExitNode"]
	if !ok {
		resultChan <- SingboxInfo{Error: "ExitNode not found"}
		return
	}

	currentNode := exitNode.Now
	allNodes := make([]string, 0)
	for _, node := range exitNode.All {
		if node != "DIRECT" && node != "REJECT" && node != "GLOBAL" {
			allNodes = append(allNodes, node)
		}
	}

	// Get delays
	delayURL := fmt.Sprintf("%s/group/ExitNode/delay?url=https%%3A%%2F%%2Fdns.google%%2F&timeout=2000", SingboxAPI)
	req2, err := http.NewRequestWithContext(ctx, "GET", delayURL, nil)
	if err != nil {
		resultChan <- SingboxInfo{
			CurrentNode: currentNode,
			AllNodes:    allNodes,
			NodeDelays:  make(map[string]int),
		}
		return
	}

	resp2, err := client.Do(req2)
	nodeDelays := make(map[string]int)
	if err == nil {
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)
		json.Unmarshal(body2, &nodeDelays)
	}

	// Set 0 for nodes without delay
	for _, node := range allNodes {
		if _, exists := nodeDelays[node]; !exists {
			nodeDelays[node] = 0
		}
	}

	resultChan <- SingboxInfo{
		CurrentNode: currentNode,
		AllNodes:    allNodes,
		NodeDelays:  nodeDelays,
	}
}

// SwitchNode switches to a different VPN exit node
func SwitchNode(nodeName string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	reqBody := fmt.Sprintf(`{"name":"%s"}`, nodeName)
	req, err := http.NewRequest("PUT", SingboxAPI+"/proxies/ExitNode",
		io.NopCloser(io.Reader(nil)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewReader([]byte(reqBody)))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("failed to switch node: status %d", resp.StatusCode)
	}

	return nil
}
