package discord

type AudioSource interface {
	ProvideFrame() ([]byte, error)
}

type VoiceSession interface {
	SetSessionId(string) error
}

type VoiceState struct {
	GuildId string
}
