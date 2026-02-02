# Super-Bot (Go Version)

High-performance Discord and Telegram bot for monitoring MikroTik, Proxmox, and managing Sing-box VPN nodes. Rewritten in Go for maximum concurrency and speed.

## Features

- **Concurrent Monitoring**: Fetches data from MikroTik, Proxmox, Sing-box, and VNPT simultaneously.
- **Node Management**: Switch Sing-box VPN exit nodes instantly.
- **Interactive Dashboard**: Real-time status in Discord (Embeds) and Telegram (Markdown).
- **High Performance**: Response time < 3s (compared to ~12s in Python version).

## Prerequisites

- **To Run**: No requirements (just a Linux OS).
- **To Build from Source**: Go 1.20+
- Access to MikroTik (SNMP enabled), Proxmox API, and Sing-box API.

## Setup

1. **Clone and Initialize**:
   ```bash
   git clone <repo-url>
   cd super-bot
   go mod tidy
   ```


2. **Configuration**:
   Copy `.env.example` to `.env` and fill in your details:
   ```bash
   cp .env.example .env
   nano .env
   ```
   **Note**: The binary reads `.env` at runtime. You do NOT need to rebuild the bot when changing configuration.

3. **Build**:
   ```bash
   go build -o super-bot cmd/both/main.go
   ```

## Usage

### Run Both Bots (Recommended)
```bash
./super-bot
```

### Run Individually
- Discord only: `go run cmd/discord/main.go`
- Telegram only: `go run cmd/telegram/main.go`

## Commands

- `/status`: Show the monitoring dashboard.
- `/ping`: Check bot latency.
- **Buttons**: Click on node buttons to switch VPN exit nodes.

## Development

- **Core Logic**: `core/` (Data fetching, API calls)
- **Discord Bot**: `bot/` & `cmd/discord/`
- **Telegram Bot**: `telegram/` & `cmd/telegram/`

## Running as a Service (Systemd)

To keep the bot running in the background and start automatically on boot:

1.  **Create a service file**:
    ```bash
    sudo nano /etc/systemd/system/super-bot.service
    ```

2.  **Paste the following content** (adjust paths as needed):
    ```ini
    [Unit]
    Description=Super-Bot Service
    After=network.target

    [Service]
    Type=simple
    User=root
    WorkingDirectory=/path/to/super-bot
    ExecStart=/path/to/super-bot/super-bot
    Restart=always
    RestartSec=10

    [Install]
    WantedBy=multi-user.target
    ```
    *Replace `/path/to/super-bot` with the actual path to your bot folder.*

3.  **Enable and Start**:
    ```bash
    sudo systemctl daemon-reload
    sudo systemctl enable super-bot
    sudo systemctl start super-bot
    ```

4.  **Check Status**:
    ```bash
    sudo systemctl status super-bot
    ```


## Releasing a New Version

The project uses GitHub Actions to automatically build and release binaries.

1.  **Commit your changes**:
    ```bash
    git add .
    git commit -m "Your changes description"
    git push origin main
    ```

2.  **Create a Tag**:
    ```bash
    git tag v1.0.0
    git push origin v1.0.0
    ```

3.  **Check GitHub**:
    - Go to the **Actions** tab to see the build progress.
    - Once finished, go to the **Releases** section to download the ready-to-use binaries.
