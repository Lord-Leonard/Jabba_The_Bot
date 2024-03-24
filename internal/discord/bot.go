package discord

import (
	"Jabba_The_Bot/internal/music"
	"Jabba_The_Bot/pkg/discord"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
)

type Bot struct {
	SessionId       string
	UserId          string
	Query           string
	commandRegistry map[string]map[string]CommandHandler
	discordService  *discord.Service
	voiceStates     map[string]*discord.VoiceState
	voiceSessions   map[string]*discord.VoiceSession
	MusicManagers   map[string]*music.Manager
}

type Command struct {
	Name           string
	Type           int8
	Description    string
	Options        []CommandOption
	CommandHandler CommandHandler
}

type CommandHandler func(ctx *CommandContext)

type CommandOption struct {
	Type        int8
	Name        string
	Description string
	Required    bool
}

func New() *Bot {
	newBot := &Bot{
		//Guilds:             []discordService.Guild{},
		voiceStates: make(map[string]*discord.VoiceState),
		//VoiceServerUpdates: make(map[string]discordService.VoiceServerUpdateData),
		//ActiveConnections:  make(map[string]*voice.Connection),
		commandRegistry: make(map[string]map[string]CommandHandler),
		discordService:  discord.NewService(),
		voiceSessions:   make(map[string]*discord.VoiceSession),
	}

	return newBot
}

func (b *Bot) RegisterSlashCommand(command *Command) error {
	if b.commandRegistry["slash"] == nil {
		b.commandRegistry["slash"] = make(map[string]CommandHandler)
	}

	b.commandRegistry["slash"][command.Name] = command.CommandHandler

	commandDTO := internal.ApplicationCommand{
		Name:        command.Name,
		Type:        command.Type,
		Description: command.Description,
	}
	if command.Options != nil && len(command.Options) > 0 {
		commandDTO.Options = &[]internal.ApplicationCommandOption{
			{
				Type:        command.Options[0].Type,
				Name:        command.Options[0].Name,
				Description: command.Options[0].Description,
				Required:    &command.Options[0].Required,
			},
		}
	}

	err := b.discordService.RegisterCommand(commandDTO)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) Run() error {
	for event := range b.discordService.Events {
		switch *event.T {
		case internal.GatewayEventNameReady:
			slog.Debug("Processing Ready Event")
			var readyData internal.ReadyData
			err := json.Unmarshal(event.D, &readyData)
			if err != nil {
				return fmt.Errorf("error parsing ready data: %w", err)
			}

			b.SessionId = readyData.SessionId
			b.UserId = readyData.User.Id

		case internal.GatewayEventNameGuildCreated:
			slog.Debug("Processing Guild Create Event")
			var guild internal.Guild
			err := json.Unmarshal(event.D, &guild)
			if err != nil {
				return fmt.Errorf("error parsing guild data: %w", err)
			}

			for _, voiceState := range *guild.VoiceStates {
				voiceState.GuildId = guild.Id
				b.voiceStates[voiceState.UserId] = voiceState
			}

		case internal.GatewayEventNameInteractionCreate:
			slog.Debug("Processing Interaction Create Event")

			var interaction internal.Interaction
			err := json.Unmarshal(event.D, &interaction)
			if err != nil {
				return fmt.Errorf("error parsing interaction data: %w", err)
			}

			interactionName := interaction.Data.Name
			handler, ok := b.commandRegistry["slash"][interactionName]
			if !ok {
				slog.Error("No handler found for interaction", "Interaction Name", interactionName)
				break
			}

			ctx := NewCommandContext(b.discordService, b, interaction)

			go handler(ctx)

		case internal.GatewayEventNameVoiceStateUpdate:
			var voiceState internal.VoiceState
			err := json.Unmarshal(event.D, &voiceState)
			if err != nil {
				return fmt.Errorf("error parsing voice state data: %w", err)
			}

			b.voiceStates[voiceState.UserId] = voiceState

			if voiceState.UserId != b.UserId {
				break
			}

			voiceSession, ok := b.voiceSessions[voiceState.GuildId]
			if !ok {
				// TODO: maybe reconnect?
				slog.Debug("No voice session found for guild", "Guild Id", voiceState.GuildId)
				break
			}
			voiceSession.SetSessionId(voiceState.SessionId)

		case internal.GatewayEventNameVoiceServerUpdate:
			var voiceServerUpdate internal.VoiceServerUpdateData
			err := json.Unmarshal(event.D, &voiceServerUpdate)
			if err != nil {
				log.Fatalln("Error parsing voice server update information", err)
			}

			voiceSession, ok := b.voiceSessions[voiceServerUpdate.GuildId]
			if !ok {
				//TODO: maybe reconnect or ?
				slog.Debug("No voice session found for guild", "Guild Id", voiceServerUpdate.GuildId)
				break
			}

			voiceSession.SetEndpoint(voiceServerUpdate.Endpoint)
			voiceSession.SetToken(voiceServerUpdate.Token)
		}
	}
	return nil
}

func (b *Bot) JoinUserChannel(userId string) (*voice.Session, error) {
	voiceState, ok := b.voiceStates[userId]
	if !ok {
		return nil, errors.New("user not in a voice channel")
	}

	voiceSession, ok := b.voiceSessions[voiceState.GuildId]
	if !ok {
		voiceSession = voice.NewSession(voiceState.GuildId, b.UserId)
		b.voiceSessions[voiceState.GuildId] = voiceSession
	}

	voiceSession.StateUpdates <- voice.StateRequestingToJoin

	err := b.discordService.UpdateVoiceState(voiceState.GuildId, voiceState.ChannelId, false, false)
	if err != nil {
		return nil, err
	}

	return voiceSession, nil
}
