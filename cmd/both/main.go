package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"super-bot/bot"
	"super-bot/core"
	"super-bot/telegram"
	"syscall"

	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	fmt.Println("🚀 Starting Super-Bot (Discord + Telegram)...")

	// --- Star Discord Bot ---
	dg, err := discordgo.New("Bot " + core.DiscordToken)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register Discord handlers
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case "status":
				bot.HandleStatusCommand(s, i)
			case "ping":
				bot.HandlePingCommand(s, i)
			}
		case discordgo.InteractionMessageComponent:
			bot.HandleButtonClick(s, i)
		}
	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("🤖 Discord: Logged in as %s", s.State.User.Username)
	})

	// Register MessageCreate handler for legacy commands
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore bot messages
		if m.Author.ID == s.State.User.ID {
			return
		}

		switch m.Content {
		case "!status":
			bot.HandleStatusMessage(s, m)
		case "!ping":
			bot.HandlePingMessage(s, m)
		}
	})

	// Open Discord connection
	if err = dg.Open(); err != nil {
		log.Fatal("Error opening Discord connection:", err)
	}
	defer dg.Close()

	// Register slash commands
	commands := []*discordgo.ApplicationCommand{
		{Name: "status", Description: "Display server dashboard"},
		{Name: "ping", Description: "Check bot latency"},
	}
	for _, cmd := range commands {
		if _, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", cmd); err != nil {
			log.Printf("Error creating Discord command %s: %v", cmd.Name, err)
		}
	}

	// --- Start Telegram Bot ---
	tgBot, err := tgbotapi.NewBotAPI(core.TelegramToken)
	if err != nil {
		log.Fatal("Error creating Telegram bot:", err)
	}
	tgBot.Debug = false
	log.Printf("🤖 Telegram: Authorized as %s", tgBot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgBot.GetUpdatesChan(u)

	// Launch Telegram polling in a goroutine
	go func() {
		for update := range updates {
			if update.Message != nil && update.Message.IsCommand() {
				// Basic auth check
				if fmt.Sprintf("%d", update.Message.Chat.ID) != core.TelegramChatID {
					// Ignore unauthorized
					continue
				}

				switch update.Message.Command() {
				case "status":
					go telegram.HandleStatusCommand(tgBot, update)
				}
			}

			if update.CallbackQuery != nil {
				go telegram.HandleButtonCallback(tgBot, update)
			}
		}
	}()

	fmt.Println("✅ All bots are running. Press CTRL+C to exit.")

	// Wait for interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("\n👋 Shutting down Super-Bot...")
}
