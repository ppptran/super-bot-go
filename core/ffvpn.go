package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Server      string `json:"server,omitempty"`
	Account     string `json:"account,omitempty"`
	EdgeSuspect bool   `json:"edge_suspect,omitempty"`
}

// FFVPNInfo aggregates edges + selection + state for the dashboards.
type FFVPNInfo struct {
	Current FFVPNEdge
	Status  FFVPNStatus
	Servers []FFVPNEdge
	Error   string
}

func ffvpnClient() *http.Client { return &http.Client{Timeout: 10 * time.Second} }

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
	if err := ffvpnHTTPError("ffvpn edges", resp); err != nil {
		resultChan <- FFVPNInfo{Error: err.Error()}
		return
	}
	var data struct {
		Current json.RawMessage `json:"current"`
		Servers []FFVPNEdge     `json:"servers"`
		Proxy   struct {
			Running     bool   `json:"running"`
			Server      string `json:"server"`
			Account     string `json:"account"`
			EdgeSuspect bool   `json:"edge_suspect"`
		} `json:"proxy"`
		Error string `json:"error"`
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
		if data.Servers[i].ServerHost != data.Servers[j].ServerHost {
			return data.Servers[i].ServerHost < data.Servers[j].ServerHost
		}
		return data.Servers[i].ServerPort < data.Servers[j].ServerPort
	})
	info := FFVPNInfo{
		Current: decodeFFVPNCurrent(data.Current, data.Servers, data.Proxy.Server),
		Status: FFVPNStatus{
			Running:     data.Proxy.Running,
			Server:      data.Proxy.Server,
			Account:     data.Proxy.Account,
			EdgeSuspect: data.Proxy.EdgeSuspect,
		},
		Servers: data.Servers,
	}
	resultChan <- info
}

// decodeFFVPNCurrent tolerates the shapes /api/edges may use for the current
// selection: a full edge object, a bare host string, or per-edge "current"
// flags. Having the host is enough for the panel; when we can match a server
// we upgrade it to the full record so country/city/port render too.
func decodeFFVPNCurrent(raw json.RawMessage, servers []FFVPNEdge, proxyServer string) FFVPNEdge {
	var current FFVPNEdge
	if len(raw) > 0 {
		switch raw[0] {
		case '{':
			_ = json.Unmarshal(raw, &current)
		case '"':
			var host string
			if json.Unmarshal(raw, &host) == nil {
				current.ServerHost = host
			}
		}
	}
	if current.ServerHost == "" {
		for _, e := range servers {
			if e.Current {
				current = e
				break
			}
		}
	}
	if current.ServerHost == "" {
		current.ServerHost = proxyServer
	}
	if current.ServerHost != "" {
		for _, e := range servers {
			if e.ServerHost == current.ServerHost {
				current = e
				break
			}
		}
	}
	return current
}

// FindFFVPNEdge resolves a country code + per-country index reference. The
// index follows the sorted Servers order, which is the same order both
// keyboards render, so a ref stays stable across refreshes.
func FindFFVPNEdge(info FFVPNInfo, countryCode string, index int) (FFVPNEdge, bool) {
	if index < 0 {
		return FFVPNEdge{}, false
	}
	n := 0
	for _, e := range info.Servers {
		if e.CountryCode != countryCode {
			continue
		}
		if n == index {
			return e, true
		}
		n++
	}
	return FFVPNEdge{}, false
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
	if err := ffvpnHTTPError("ffvpn select", resp); err != nil {
		return err
	}
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

// helpers

// ffvpnHTTPError turns a non-2xx reply into an error carrying the status
// and a body snippet. Without this, endpoints that answer 401/500 with
// plain text (or an empty body) look like a successful call.
func ffvpnHTTPError(op string, resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return fmt.Errorf("%s: HTTP %d: %s", op, resp.StatusCode, msg)
}
