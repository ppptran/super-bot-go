package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"super-bot/bot"
	"super-bot/core"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	fmt.Println("🚀 Starting Discord Bot...")

	// Create Discord session
	dg, err := discordgo.New("Bot " + core.DiscordToken)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register handlers
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			// Handle slash commands
			switch i.ApplicationCommandData().Name {
			case "status":
				bot.HandleStatusCommand(s, i)
			case "ping":
				bot.HandlePingCommand(s, i)
			}
		case discordgo.InteractionMessageComponent:
			// Handle button clicks
			bot.HandleButtonClick(s, i)
		}
	})

	// Register ready handler
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("🤖 Discord Bot logged in as %s\n", s.State.User.Username)
		fmt.Println("📊 Ready to serve dashboard!")
	})

	// Open connection
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}
	defer dg.Close()

	// Register slash commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "status",
			Description: "Display server dashboard",
		},
		{
			Name:        "ping",
			Description: "Check bot latency",
		},
	}

	for _, cmd := range commands {
		_, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create command %s: %v", cmd.Name, err)
		}
	}

	fmt.Println("✅ Discord Bot is running. Press CTRL+C to exit.")

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("\n👋 Shutting down Discord Bot...")
}
