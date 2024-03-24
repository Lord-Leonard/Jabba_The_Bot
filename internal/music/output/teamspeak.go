package output

import (
	"Jabba_The_Bot/internal/music/provider"
	"Jabba_The_Bot/pkg/teamspeak"
)

type TeamSpeak struct {
	client *teamspeak.Client
}

func NewTeamSpeak(client *teamspeak.Client) *TeamSpeak {
	return &TeamSpeak{client: client}
}

func (t *TeamSpeak) PlayOpus(source provider.AudioStream) error {
	return t.client.SendOpusStream(source)
}
