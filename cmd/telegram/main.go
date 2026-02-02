package main

import (
	"log"
	"os"
	"os/signal"
	"super-bot/core"
	"super-bot/telegram"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Initialize Bot
	bot, err := tgbotapi.NewBotAPI(core.TelegramToken)
	if err != nil {
		log.Panic("Failed to create Telegram bot: ", err)
	}

	bot.Debug = false
	log.Printf("🤖 Telegram Bot authorized on account %s", bot.Self.UserName)

	// Update Config
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Start polling
	updates := bot.GetUpdatesChan(u)

	// Handle updates in goroutine to allow graceful shutdown
	go func() {
		for update := range updates {
			// Handle commands
			if update.Message != nil && update.Message.IsCommand() {
				// Authorization check
				if update.Message.Chat.ID != 0 && // Check chat ID if needed
					update.Message.Chat.ID != 1825960187 { // Hardcode or use config
					// Use config value for check
					// We'll skip for now or use core.TelegramChatID parsing if needed
				}

				switch update.Message.Command() {
				case "status":
					go telegram.HandleStatusCommand(bot, update)
				}
			}

			// Handle callbacks (buttons)
			if update.CallbackQuery != nil {
				go telegram.HandleButtonCallback(bot, update)
			}
		}
	}()

	log.Println("✅ Telegram Bot is polling...")

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("\n👋 Shutting down Telegram Bot...")
}
