package output

import (
	"Jabba_The_Bot/internal/music/provider"
	discord2 "Jabba_The_Bot/pkg/discord"
)

type discordVoicePlayer interface {
	PlayTest(source discord2.AudioSource) error
}

type Discord struct {
	player discordVoicePlayer
}

func NewDiscord(player discordVoicePlayer) *Discord {
	return &Discord{player: player}
}

func (d *Discord) PlayOpus(source provider.AudioStream) error {
	return d.player.PlayTest(source)
}
