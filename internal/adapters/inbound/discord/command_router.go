package discord

import "log/slog"

// CommandRouter is a scaffold for the planned Discord inbound adapter.
// It will eventually map slash/chat interactions into core MusicCommands.
type CommandRouter struct{}

func NewCommandRouter() *CommandRouter {
	return &CommandRouter{}
}

func (r *CommandRouter) Register() {
	slog.Info("discord inbound adapter scaffold initialized")
}
