# FFVPN integration (super-bot fork)

Adds `/ffvpn` command + edge-selection buttons to Telegram/Discord for
managing the FFVPN container (github.com/…/TuiFFVpn) — list its MASQUE
edges, switch the selected server, watch proxy state. sing-box keeps ONE
outbound (`FF-VPN` socks → ffvpn :1080); server switching is a
control-plane call to ffvpn, not a sing-box selector change.

## New files

- `core/ffvpn.go` — edge API client (list/select/status)
- `telegram/ffvpn_handlers.go` — `/ffvpn` command + inline keyboard
- `bot/ffvpn_handlers.go` + `bot/ffvpn_components.go` — Discord slash
  command + embed/buttons
- `cmd/*/main.go` — ffvpn dispatch wired (command + callback routing)

## Configuration

Add to `.env`:

```
FFVPN_API=http://192.168.100.85:9090   # ffvpn web console (host A)
FFVPN_KEY=<FXVPN_INGEST_KEY value>     # shared secret
```

## Dispatch wiring (already applied in this fork)

Discord: `/ffvpn` slash command case + `ffvpn*` CustomID prefix routing to
`HandleFFVPNButtonClick`. Telegram: `ffvpn` command case + `ffvpn`-prefixed
callback routing to `HandleFFVPNButtonCallback`. Discord slash commands
also need the `/ffvpn` entry registered in the command-registration code
(same place `/status` is registered).

## Traffic path

  devices → sing-box mixed-in (:2080) → FF-VPN socks outbound
          → ffvpn :1080 → current MASQUE edge

Switching an edge in the bot restarts ffvpn's SOCKS listener on the new
edge in milliseconds; sing-box's outbound stays connected — devices never
reconfigure.
