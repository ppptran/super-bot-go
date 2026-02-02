package core

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Configuration variables
var (
	// Telegram Config
	TelegramToken  string
	TelegramChatID string

	// Discord Config
	DiscordToken     string
	DiscordChannelID string

	// Infrastructure Config
	PVEHost       string
	PVEUser       string
	PVETokenName  string
	PVETokenValue string

	MikroTikIP    string
	SNMPCommunity string
	PPPoEIndex    string

	SingboxAPI string
)

func init() {
	// Attempt to load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found. Using environment variables or empty defaults.")
	}

	// Telegram
	TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")

	// Discord
	DiscordToken = os.Getenv("DISCORD_TOKEN")
	DiscordChannelID = os.Getenv("DISCORD_CHANNEL_ID")

	// Proxmox
	PVEHost = os.Getenv("PVE_IP")
	PVEUser = os.Getenv("PVE_USER")
	PVETokenName = os.Getenv("PVE_TOKEN_NAME")
	PVETokenValue = os.Getenv("PVE_TOKEN_VALUE")

	// MikroTik
	MikroTikIP = os.Getenv("MIKROTIK_IP")
	SNMPCommunity = os.Getenv("SNMP_COMMUNITY")
	PPPoEIndex = os.Getenv("PPPOE_INDEX")

	// Sing-box
	SingboxAPI = os.Getenv("SINGBOX_API")

	// Set defaults if needed (optional)
	if SingboxAPI == "" {
		SingboxAPI = "http://127.0.0.1:9090"
	}
}
