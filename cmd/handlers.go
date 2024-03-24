package main

import (
	"Jabba_The_Bot/internal/discord"
	"Jabba_The_Bot/internal/music"
	"Jabba_The_Bot/internal/music/output"
	"Jabba_The_Bot/internal/music/provider"
	"encoding/json"
	"log/slog"
)

type Handlers struct {
	youtube *provider.YouTubeProvider
}

func NewHandlers(yt *provider.YouTubeProvider) *Handlers {
	return &Handlers{youtube: yt}
}

func (h *Handlers) ping(ctx *discord.CommandContext) {
	err := ctx.Reply("pong")
	if err != nil {
		slog.Error("Error replying to ping", "Error", err)
		return
	}

	_, _ = ctx.EditOriginalInteractionResponse("pong 3")
}

func (h *Handlers) play(ctx *discord.CommandContext) {
	bot := ctx.DiscordBot
	if len(ctx.RawInteraction.Data.Options) == 0 {
		slog.Error("No options provided in play command")
		return
	}

	var query string
	err := json.Unmarshal(ctx.RawInteraction.Data.Options[0].Value, &query)
	if err != nil {
		slog.Error("Error unmarshalling query", "Error", err)
		// Fallback to string cast if unmarshal fails, though it shouldn't for a string option
		query = string(ctx.RawInteraction.Data.Options[0].Value)
	}

	voiceSession, err := bot.JoinUserChannel(ctx.UserId)
	if err != nil {
		slog.Error("Error joining user channel", "Error", err)
		ctx.Reply("Nutzer muss sich in einem Sprachkanal befinden!")
		return
	}

	ctx.Reply("Connecting ...")

	voiceSession.WaitReady()

	manager, ok := ctx.DiscordBot.MusicManagers[voiceSession.GetSessionId()]
	if !ok {
		manager = music.NewManager(
			*provider.NewYouTubeProvider(),
			output.NewDiscord(voiceSession),
		)
		if ctx.DiscordBot.MusicManagers == nil {
			ctx.DiscordBot.MusicManagers = make(map[string]*music.Manager)
		}
		ctx.DiscordBot.MusicManagers[voiceSession.GetSessionId()] = manager
	}

	go func() {
		err := manager.Start(query)
		if err != nil {
			slog.Error("Error starting music manager", "Error", err)
		}
	}()
}
