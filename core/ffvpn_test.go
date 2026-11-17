package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFFVPNInfoDecodesEdgeSuspect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/edges" {
			t.Errorf("path = %q, want /api/edges", r.URL.Path)
		}
		if got := r.Header.Get("X-Ingest-Key"); got != "test-key" {
			t.Errorf("X-Ingest-Key = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
			"current": {"country_code":"SG","country_name":"Singapore","city_name":"Singapore","server_host":"sg1.example","server_port":443},
			"servers": [
				{"country_code":"US","country_name":"United States","city_name":"Ashburn","server_host":"us1.example","server_port":443},
				{"country_code":"SG","country_name":"Singapore","city_name":"Singapore","server_host":"sg1.example","server_port":443}
			],
			"proxy": {"running": true, "server": "sg1.example", "account": "acct", "edge_suspect": true}
		}`)
	}))
	defer srv.Close()
	t.Setenv("FFVPN_API", srv.URL)
	t.Setenv("FFVPN_KEY", "test-key")

	ch := make(chan FFVPNInfo, 1)
	GetFFVPNInfo(context.Background(), ch)
	info := <-ch

	if info.Error != "" {
		t.Fatalf("unexpected error: %s", info.Error)
	}
	if !info.Status.EdgeSuspect {
		t.Fatal("edge_suspect was not decoded")
	}
	if info.Current.ServerHost != "sg1.example" {
		t.Fatalf("current = %q, want sg1.example", info.Current.ServerHost)
	}
	if got := len(info.Servers); got != 2 {
		t.Fatalf("servers = %d, want 2", got)
	}
	if info.Servers[0].CountryName != "Singapore" {
		t.Fatalf("servers not sorted by country: first = %q", info.Servers[0].CountryName)
	}
}

func TestGetFFVPNInfoHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("FFVPN_API", srv.URL)
	t.Setenv("FFVPN_KEY", "test-key")

	ch := make(chan FFVPNInfo, 1)
	GetFFVPNInfo(context.Background(), ch)
	if info := <-ch; info.Error == "" {
		t.Fatal("expected an error for HTTP 500")
	}
}

func TestSwitchFFVPNEdgeSuccess(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/edges/select" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	t.Setenv("FFVPN_API", srv.URL)
	t.Setenv("FFVPN_KEY", "test-key")

	edge := FFVPNEdge{
		CountryCode: "SG",
		CountryName: "Singapore",
		CityName:    "Singapore",
		ServerHost:  "sg1.example",
		ServerPort:  443,
	}
	if err := SwitchFFVPNEdge(edge); err != nil {
		t.Fatalf("SwitchFFVPNEdge: %v", err)
	}
	if got["server_host"] != "sg1.example" || got["server_port"] != float64(443) {
		t.Fatalf("payload = %#v", got)
	}
}

func TestSwitchFFVPNEdgeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	defer srv.Close()
	t.Setenv("FFVPN_API", srv.URL)
	t.Setenv("FFVPN_KEY", "test-key")

	if err := SwitchFFVPNEdge(FFVPNEdge{ServerHost: "sg1.example"}); err == nil {
		t.Fatal("expected an error for HTTP 401")
	}
}
