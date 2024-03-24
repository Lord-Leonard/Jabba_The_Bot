package discord

import "Jabba_The_Bot/pkg/discord/internal"

type Service struct {
}

type ApplicationCommandOption = internal.ApplicationCommandOption

type ApplicationCommand struct {
	Id                       string                      `json:"id"`
	Type                     int8                        `json:"type,omitempty"`
	ApplicationId            string                      `json:"application_id"`
	GuildId                  *string                     `json:"guild_id,omitempty"`
	Name                     string                      `json:"name"`
	Description              string                      `json:"description"`
	Options                  *[]ApplicationCommandOption `json:"options,omitempty"`
	DefaultMemberPermissions *string                     `json:"default_member_permissions,omitempty"`
	DMPermission             *bool                       `json:"dm_permission,omitempty"`
	DefaultPermission        *bool                       `json:"default_permission,omitempty"`
	NSFW                     *bool                       `json:"nsfw,omitempty"`
	Version                  *string                     `json:"version,omitempty"`
}

func NewService() *Service {
	return &Service{}
}
