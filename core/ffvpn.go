package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// FFVPN edge control API client.
//
// ffvpn (the Firefox VPN container) owns the server list and runs the
// SOCKS5 listener sing-box chains through. Server switching is a
// control-plane call to ffvpn's edge API — sing-box's outbound never
// changes. Endpoints (auth: X-Ingest-Key header, the FXVPN_INGEST_KEY
// value):
//
//	GET  /api/edges         — full server list + current selection + state
//	POST /api/edges/select  — switch server
//	GET  /api/edges/status  — current selection + proxy state + suspect flag

// ffvpnAPI / ffvpnKey read the env AT CALL TIME. They must not be
// captured in package-level vars: godotenv.Load() runs in config.go's
// init(), and package-level var initializers execute BEFORE init()
// functions — capturing at var-init would yield empty values whenever
// the vars are supplied through a .env file.
func ffvpnAPI() string {
	api := os.Getenv("FFVPN_API")
	return strings.TrimRight(api, "/")
}

func ffvpnKey() string {
	return os.Getenv("FFVPN_KEY")
}

func ffvpnConfigured() bool {
	return ffvpnAPI() != "" && ffvpnKey() != ""
}

// FFVPNEdge is one selectable MASQUE edge host.
type FFVPNEdge struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	CityName    string `json:"city_name"`
	ServerHost  string `json:"server_host"`
	ServerPort  int    `json:"server_port"`
	Current     bool   `json:"current"`
}

// FFVPNStatus is the proxy state snapshot.
type FFVPNStatus struct {
	Running     bool   `json:"running"`
	Listen      string `json:"listen,omitempty"`
	Server      string `json:"server,omitempty"`
	Account     string `json:"account,omitempty"`
	EdgeSuspect bool   `json:"edge_suspect,omitempty"`
}

// FFVPNInfo aggregates edges + selection + state for the dashboards.
type FFVPNInfo struct {
	Current  FFVPNEdge
	Status   FFVPNStatus
	Servers  []FFVPNEdge
	EdgeList string // formatted for display, filled by GetFFVPNInfo
	Error    string
}

func ffvpnClient() *http.Client { return &http.Client{Timeout: 10 * time.Second} }

func ffvpnLabel(e FFVPNEdge) string {
	return fmt.Sprintf("%s/%s — %s:%d", e.CountryName, e.CityName, e.ServerHost, e.ServerPort)
}

// GetFFVPNInfo fetches edges + status; async channel style like GetSingboxInfo.
func GetFFVPNInfo(ctx context.Context, resultChan chan<- FFVPNInfo) {
	defer close(resultChan)
	if !ffvpnConfigured() {
		resultChan <- FFVPNInfo{Error: "FFVPN_API / FFVPN_KEY not configured in .env"}
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ffvpnAPI()+"/api/edges", nil)
	if err != nil {
		resultChan <- FFVPNInfo{Error: err.Error()}
		return
	}
	req.Header.Set("X-Ingest-Key", ffvpnKey())
	resp, err := ffvpnClient().Do(req)
	if err != nil {
		resultChan <- FFVPNInfo{Error: err.Error()}
		return
	}
	defer resp.Body.Close()
	var data struct {
		Current   FFVPNEdge   `json:"current"`
		Servers   []FFVPNEdge `json:"servers"`
		Proxy     struct {
			Running bool   `json:"running"`
			Server  string `json:"server"`
			Account string `json:"account"`
		} `json:"proxy"`
		EdgeCount int    `json:"edge_count"`
		Error     string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		resultChan <- FFVPNInfo{Error: "parse edges: " + err.Error()}
		return
	}
	if data.Error != "" {
		resultChan <- FFVPNInfo{Error: data.Error}
		return
	}
	sort.Slice(data.Servers, func(i, j int) bool {
		if data.Servers[i].CountryName != data.Servers[j].CountryName {
			return data.Servers[i].CountryName < data.Servers[j].CountryName
		}
		if data.Servers[i].CityName != data.Servers[j].CityName {
			return data.Servers[i].CityName < data.Servers[j].CityName
		}
		return data.Servers[i].ServerHost < data.Servers[j].ServerHost
	})
	info := FFVPNInfo{
		Current: data.Current,
		Status: FFVPNStatus{
			Running: data.Proxy.Running,
			Server:  data.Proxy.Server,
			Account: data.Proxy.Account,
		},
		Servers: data.Servers,
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📍 Current: %s\n", ffvpnLabel(data.Current)))
	sb.WriteString(fmt.Sprintf("🔗 Proxy: %s on %s (account %s)\n",
		onOff(info.Status.Running), valOr(info.Status.Server, "—"), valOr(info.Status.Account, "—")))
	if info.Status.EdgeSuspect {
		sb.WriteString("⚠️ This edge is rejecting tunnels server-side — pick another server.\n")
	}
	sb.WriteString(fmt.Sprintf("🌐 %d edges available", len(data.Servers)))
	info.EdgeList = sb.String()
	resultChan <- info
}

// SwitchFFVPN selects an edge by host name (the button payload).
func SwitchFFVPN(host string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ch := make(chan FFVPNInfo, 1)
	GetFFVPNInfo(ctx, ch)
	info := <-ch
	if info.Error != "" {
		return fmt.Errorf("ffvpn list: %s", info.Error)
	}
	for _, e := range info.Servers {
		if e.ServerHost == host {
			return SwitchFFVPNEdge(e)
		}
	}
	return fmt.Errorf("unknown ffvpn edge: %s", host)
}

// SwitchFFVPNEdge posts the full select payload for an edge.
func SwitchFFVPNEdge(e FFVPNEdge) error {
	if !ffvpnConfigured() {
		return fmt.Errorf("FFVPN_API / FFVPN_KEY not configured in .env")
	}
	body, _ := json.Marshal(map[string]any{
		"country_code": e.CountryCode,
		"country_name": e.CountryName,
		"city_name":    e.CityName,
		"server_host":  e.ServerHost,
		"server_port":  e.ServerPort,
	})
	req, err := http.NewRequest(http.MethodPost,
		ffvpnAPI()+"/api/edges/select", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Ingest-Key", ffvpnKey())
	resp, err := ffvpnClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		Error string `json:"error"`
		Note  string `json:"note"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Error != "" {
		return fmt.Errorf("ffvpn select: %s", out.Error)
	}
	return nil
}

// GetFFVPNStatus fetches /api/edges/status for quick checks.
func GetFFVPNStatus(ctx context.Context) (FFVPNStatus, error) {
	var st FFVPNStatus
	if !ffvpnConfigured() {
		return st, fmt.Errorf("FFVPN_API / FFVPN_KEY not configured in .env")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		ffvpnAPI()+"/api/edges/status", nil)
	if err != nil {
		return st, err
	}
	req.Header.Set("X-Ingest-Key", ffvpnKey())
	resp, err := ffvpnClient().Do(req)
	if err != nil {
		return st, err
	}
	defer resp.Body.Close()
	var data struct {
		Selected struct {
			ServerHost string `json:"server_host"`
		} `json:"selected"`
		Proxy FFVPNStatus `json:"proxy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return st, err
	}
	st = data.Proxy
	if st.Server == "" {
		st.Server = data.Selected.ServerHost
	}
	return st, nil
}

// helpers
func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func onOff(b bool) string {
	if b {
		return "running"
	}
	return "stopped"
}

func valOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
