package teamspeak

import (
	"Jabba_The_Bot/internal/music/playback"
	"Jabba_The_Bot/pkg/teamspeak"
)

type AudioAdapter struct {
	client *teamspeak.Client
}

func NewAudioAdapter(client *teamspeak.Client) *AudioAdapter {
	return &AudioAdapter{client: client}
}

func (a *AudioAdapter) Consume(source playback.AudioStream) error {
	return a.client.SendOpusStream(source)
}
