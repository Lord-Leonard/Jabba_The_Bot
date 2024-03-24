package discord

import (
	"Jabba_The_Bot/pkg/discord/internal"
)

type CommandContext struct {
	RawInteraction internal.Interaction

	DiscordService *internal.Service
	DiscordBot     *Bot

	interactionId    string
	interactionToken string

	UserId  string
	GuildId string
}

func NewCommandContext(discord *internal.Service, bot *Bot, rawInteractionCreateData internal.Interaction) *CommandContext {
	return &CommandContext{
		RawInteraction:   rawInteractionCreateData,
		DiscordService:   discord,
		DiscordBot:       bot,
		interactionId:    rawInteractionCreateData.Id,
		interactionToken: rawInteractionCreateData.Token,
		UserId:           rawInteractionCreateData.Member.User.Id,
		GuildId:          rawInteractionCreateData.GuildId,
	}
}

func (ctx *CommandContext) Reply(message string) error {
	payload := internal.InteractionResponse{
		Type: internal.InteractionTypeChannelMessageWithSource,
		Data: &internal.InteractionCallbackDataMessage{
			Content: message,
			Flags:   0,
		},
	}

	_, err := ctx.CreateInteractionResponse(payload, false)
	return err
}

func (ctx *CommandContext) CreateInteractionResponse(payload internal.InteractionResponse, withResponse bool) ([]byte, error) {
	return ctx.DiscordService.CreateInteractionResponse(ctx.interactionId, ctx.interactionToken, payload, withResponse)
}

func (ctx *CommandContext) GetOriginalInteractionResponse() (*internal.Message, error) {
	return ctx.DiscordService.GetOriginalInteractionResponse(ctx.interactionToken)
}

func (ctx *CommandContext) EditOriginalInteractionResponse(message string) (*internal.Message, error) {
	payload := internal.WebhookMessageEdit{
		Content: message,
	}
	return ctx.DiscordService.EditOriginalInteractionResponse(ctx.interactionToken, payload)
}

func (ctx *CommandContext) DeleteOriginalInteractionResponse() error {
	return ctx.DiscordService.DeleteOriginalInteractionResponse(ctx.interactionToken)
}

func (ctx *CommandContext) CreateFollowupMessage(payload internal.InteractionFollowupMessage) (*internal.Message, error) {
	return ctx.DiscordService.CreateFollowupMessage(ctx.interactionToken, payload)
}
