package main

import (
	"Jabba_The_Bot/internal/discord"
	"Jabba_The_Bot/internal/music/provider"
	"log"
	"log/slog"
	"os"
	"runtime/trace"

	"github.com/joho/godotenv"
)

func main() {
	loadDotenv()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	f, _ := os.Create("trace.out")
	trace.Start(f)
	defer trace.Stop()

	discordBot := discord.New()
	youtubeProvider, _ := provider.NewYouTubeProvider(nil)
	handlers := NewHandlers(youtubeProvider)

	pingCommand := &discord.Command{
		Name:           "ping",
		Type:           1,
		Description:    "Ping - Pong",
		CommandHandler: handlers.ping,
	}

	playCommand := &discord.Command{
		Name:        "play",
		Type:        1,
		Description: "Play music from YouTube",
		Options: []discord.CommandOption{
			{
				Name:        "query",
				Type:        3,
				Description: "The search query or YouTube URL",
				Required:    true,
			},
		},
		CommandHandler: handlers.play,
	}

	discordBot.RegisterSlashCommand(pingCommand)
	discordBot.RegisterSlashCommand(playCommand)

	err := discordBot.Run()
	if err != nil {
		log.Fatalln("Error running bot:", err)
	}
}

func loadDotenv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Unable to load config")
	}
}
