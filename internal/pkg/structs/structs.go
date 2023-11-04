package structs

import (
	"encoding/json"
	"time"
)

type GatewayResponse struct {
	URL string `json:"url"`
}

type WebsocketMessage struct {
	OP int              `json:"op"`
	D  *json.RawMessage `json:"d"`
	S  *int             `json:"s,omitempty"`
	T  *string          `json:"t,omitempty"`
}

type HelloData struct {
	Heartbeat_interval int `json:"heartbeat_interval"`
}

type IdentifyData struct {
	Token           string                       `json:"token"`
	Properties      IdentifyConnectionProperties `json:"properties"`
	Compress        *bool                        `json:"compress,omitempty"`
	Large_threshold *int                         `json:"large_threshold,omitempty"`
	Shard           *[2]int                      `json:"shard,omitempty"`
	Presence        *json.RawMessage             `json:"presence,omitempty"`
	Intents         int                          `json:"intents"`
}

type IdentifyConnectionProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type ReadyData struct {
	V      int  `json:"v"`
	User   User `json:"user"`
	Guilds []struct {
		Id          string `json:"id"`
		Unavailible bool   `json:"unavailible"`
	} `json:"guilds"`
	SessionId        string `json:"session_id"`
	ResumeGatewayUrl string `json:"resume_gateway_url"`
	Shard            *[2]int
	Application      struct {
		Id    string `json:"id"`
		Flags string `json:"flags"`
	}
}

type Heartbeat struct {
	OP int  `json:"op"`
	D  *int `json:"d"`
}

type User struct {
	Id               string  `json:"id"`
	Username         string  `json:"username"`
	Discriminator    string  `json:"discriminator"`
	GlobalName       *string `json:"global_name"`
	Avatar           *string `json:"avatar"`
	Bot              bool    `json:"bot"`
	System           bool    `json:"system"`
	MfaEnabled       bool    `json:"mfa_enabled"`
	Banner           *string `json:"banner"`
	AccentColor      *int    `json:"accent_color"`
	Locale           string  `json:"locale"`
	Verified         bool    `json:"verified"`
	Email            *string `json:"email"`
	Flags            int     `json:"flags"`
	PremiumType      int     `json:"premium_type"`
	PublicFlags      int     `json:"public_flags"`
	AvatarDecoration *string `json:"avatar_decoration"`
}

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

type ApplicationCommandOption struct {
	Type         int8                              `json:"type"`
	Name         string                            `json:"name"`
	Description  string                            `json:"description"`
	Required     *bool                             `json:"required"`
	Choices      *[]ApplicationCommandOptionChoice `json:"choices"`
	Options      *[]ApplicationCommandOption       `json:"options"`
	ChannelTypes *[]int8                           `json:"channel_types"`
	MinValue     *json.RawMessage                  `json:"min_value"`
	MaxValue     *json.RawMessage                  `json:"max_value"`
	MinLength    *int16                            `json:"min_length"`
	MaxLength    *int16                            `json:"max_length"`
	Autocomplete *bool                             `json:"autocomplete"`
}

type ApplicationCommandOptionChoice struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

type InteractionCreateData struct {
	Id            string                 `json:"id"`
	ApplicationId string                 `json:"application_id"`
	Type          int                    `json:"type"`
	Data          ApplicationCommandData `json:"data,omitempty"`
	GuildId       string                 `json:"guild_id"`
	Guild         struct {
		Locale   string   `json:"locale"`
		Id       string   `json:"id"`
		Features []string `json:"features"`
	} `json:"guild"`
	Channel           Channel       `json:"channel,omitempty"`
	ChannelId         string        `json:"channel_id,omitempty"`
	Member            GuildMember   `json:"member,omitempty"`
	User              User          `json:"user,omitempty"`
	Token             string        `json:"token"`
	Version           int           `json:"version"`
	Message           Message       `json:"message"`
	AppPermissions    string        `json:"app_permissions,omitempty"`
	Locale            string        `json:"locale,omitempty"`
	GuildLocale       string        `json:"guild_locale,omitempty"`
	Entitlements      []Entitlement `json:"entitlements"`
	EntitlementSkuIds []string      `json:"entitlement_sku_ids"`
}

type GuildMember struct {
	User                       User       `json:"user,omitempty"`
	Nick                       *string    `json:"nick,omitempty"`
	Avatar                     *string    `json:"avatar,omitempty"`
	Roles                      []string   `json:"roles"`
	JoinedAt                   time.Time  `json:"joined_at"`
	PremiumSince               *time.Time `json:"premium_since"`
	Deaf                       bool       `json:"deaf"`
	Mute                       bool       `json:"mute"`
	Flags                      int        `json:"flags"`
	Pending                    bool       `json:"pending,omitempty"`
	Permissions                string     `json:"permissions,omitempty"`
	CommunicationDisabledUntil *time.Time `json:"communication_disabled_until,omitempty"`
}

type Entitlement struct {
	Id            string    `json:"id"`
	SkuId         string    `json:"sku_id"`
	ApplicationId string    `json:"application_id"`
	UserId        string    `json:"user_id,omitempty"`
	Type          int       `json:"type"`
	Deleted       bool      `json:"deleted"`
	StartsAt      time.Time `json:"starts_at,omitempty"`
	EndsAt        time.Time `json:"ends_at,omitempty"`
	GuildId       string    `json:"guild_id,omitempty"`
}

type ApplicationCommandData struct {
	Id       string                                  `json:"id"`
	Name     string                                  `json:"name"`
	Type     int                                     `json:"type"`
	Resolved ResolvedData                            `json:"resolved,omitempty"`
	Options  ApplicationCommandInteractionDataOption `json:"options,omitempty"`
	GuildId  string                                  `json:"guild_id,omitempty"`
	TargetId string                                  `json:"target_id,omitempty"`
}

type ResolvedData struct {
	Users       []string `json:"users,omitempty"`
	Members     []string `json:"members,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Messages    []string `json:"messages,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type ApplicationCommandInteractionDataOption struct {
	Name    string                                   `json:"name"`
	Type    string                                   `json:"type"`
	Value   json.RawMessage                          `json:"value,omitempty"`
	Options *ApplicationCommandInteractionDataOption `json:"options,omitempty"`
	Focused bool                                     `json:"focused,omitempty"`
}

type Channel struct {
	Id                            string          `json:"id"`
	Type                          int             `json:"type"`
	GuildId                       string          `json:"guild_id,omitempty"`
	Positions                     int             `json:"positions,omitempty"`
	PermissionsOverwrites         []Overwrite     `json:"permissions_overwrites,omitempty"`
	Name                          string          `json:"name,omitempty"`
	Topic                         string          `json:"topic,omitempty"`
	NSFW                          bool            `json:"nsfw,omitempty"`
	LastMessageId                 string          `json:"last_message_id,omitempty"`
	Bitrate                       int             `json:"bitrate,omitempty"`
	UserLimit                     int             `json:"user_limit,omitempty"`
	RateLimitPerUser              int             `json:"rate_limit_per_user,omitempty"`
	Recipients                    int             `json:"recipients,omitempty"`
	Icon                          string          `json:"icon,omitempty"`
	OwnerId                       string          `json:"owner_id,omitempty"`
	ApplicationId                 string          `json:"application_id,omitempty"`
	Managed                       bool            `json:"managed,omitempty"`
	ParentId                      string          `json:"parent_id,omitempty"`
	LastPinTimestamp              time.Time       `json:"last_pin_timestamp,omitempty"`
	RTCRegion                     string          `json:"rtc_region,omitempty"`
	VideoQualityMode              int             `json:"video_quality_mode,omitempty"`
	MessageCount                  int             `json:"message_count,omitempty"`
	MemberCount                   int             `json:"member_count,omitempty"`
	ThreadMetadata                ThreadMetadata  `json:"thread_metadata,omitempty"`
	Member                        ThreadMember    `json:"member,omitempty"`
	DefaultAutoArchiveDuration    int             `json:"default_auto_archive_duration,omitempty"`
	Permissions                   string          `json:"permissions,omitempty"`
	Flags                         int             `json:"flags,omitempty"`
	TotalMessagesSent             int             `json:"total_messages_sent,omitempty"`
	AvailableTags                 []FormTag       `json:"available_tags,omitempty"`
	AppliedTags                   []string        `json:"applied_tags,omitempty"`
	DefaultReactionEmoji          DefaultReaction `json:"default_reaction_emoji,omitempty"`
	DefaultThreadRateLimitPerUser int             `json:"default_thread_rate_limit_per_user,omitempty"`
	DefaultSortOrder              int             `json:"default_sort_order,omitempty"`
	DefaultForumLayout            int             `json:"default_forum_layout,omitempty"`
}

type Overwrite struct {
	Id    string `json:"id"`
	Type  int    `json:"type"`
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type ThreadMetadata struct {
	Archived            bool       `json:"archived"`
	AutoArchiveDuration int        `json:"auto_archive_duration"`
	ArchiveTimestamp    time.Time  `json:"archive_timestamp"`
	Locked              bool       `json:"locked"`
	Invitable           bool       `json:"invitable,omitempty"`
	CreateTimestamp     *time.Time `json:"create_timestamp,omitempty"`
}

type ThreadMember struct {
	Id            string      `json:"id,omitempty"`
	UserId        string      `json:"user_id,omitempty"`
	JoinTimestemp time.Time   `json:"join_timestamp"`
	Flags         int         `json:"flags"`
	Member        GuildMember `json:"member,omitempty"`
}

type FormTag struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Moderated bool   `json:"moderated"`
	EmojiId   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
}

type DefaultReaction struct {
	EmojiId   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
}

type InteractionResponse struct {
	Type int             `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type InteractionCallbackDataMessage struct {
	TTS           bool            `json:"tts"`
	Content       string          `json:"content,omitempty"`
	Embeds        []Embed         `json:"embeds"`
	AllowMentions AllowMentions   `json:"allow_mentions,omitempty"`
	Flags         int             `json:"flags,omitempty"`
	Components    json.RawMessage `json:"components,omitempty"`
	Attachments   []Attachment    `json:"attachments,omitempty"`
}

type Embed struct {
	Title       string         `json:"title,omitempty"`
	Type        string         `json:"type,omitempty"`
	Description string         `json:"description,omitempty"`
	URL         string         `json:"url,omitempty"`
	Timestamp   time.Time      `json:"timestamp,omitempty"`
	Color       int            `json:"color,omitempty"`
	Footer      EmbedFooter    `json:"footer,omitempty"`
	Image       EmbedImage     `json:"image,omitempty"`
	Thumbnail   EmbedThumbnail `json:"thumbnail,omitempty"`
	Video       EmbedVideo     `json:"video,omitempty"`
	Provider    EmbedProvider  `json:"provider,omitempty"`
	Author      EmbedAuthor    `json:"author,omitempty"`
	Fields      []EmbedField   `json:"fields,omitempty"`
}

type EmbedFooter struct {
	Text         string `json:"text"`
	IconUrl      string `json:"icon_url,omitempty"`
	ProxyIconUrl string `json:"proxy_icon_url,omitempty"`
}

type EmbedImage struct {
	Url      string `json:"url"`
	ProxyUrl string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type EmbedThumbnail struct {
	Url      string `json:"url"`
	ProxyUrl string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type EmbedVideo struct {
	Url      string `json:"url"`
	ProxyUrl string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type EmbedProvider struct {
	Name string `json:"name,omitempty"`
	Url  string `json:"url,omitempty"`
}

type EmbedAuthor struct {
	Name         string `json:"name"`
	Url          string `json:"url,omitempty"`
	IconUrl      string `json:"icon_url,omitempty"`
	ProxyIconUrl string `json:"proxy_icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type AllowMentions struct {
	Parse       []string `json:"parse"`
	Roles       []string `json:"roles,omitempty"`
	Users       []string `json:"users,omitempty"`
	RepliedUser bool     `json:"replied_user,omitempty"`
}

type Attachment struct {
	Id           string  `json:"id"`
	Filename     string  `json:"filename"`
	Description  string  `json:"description,omitempty"`
	ContentType  string  `json:"content_type,omitempty"`
	Size         int     `json:"size"`
	Url          string  `json:"url"`
	ProxyUrl     string  `json:"proxy_url"`
	Height       int     `json:"height,omitempty"`
	Width        int     `json:"width,omitempty"`
	Ephemeral    bool    `json:"ephemeral,omitempty"`
	DurationSecs float32 `json:"duration_secs,omitempty"`
	Waveform     string  `json:"waveform,omitempty"`
	Flags        int     `json:"flags,omitempty"`
}

type Message struct {
	Id                   string               `json:"id"`
	ChannelId            string               `json:"channel_id"`
	Author               User                 `json:"author"`
	Content              string               `json:"content"`
	Timestamp            time.Time            `json:"timestamp"`
	EditedTimestamp      time.Time            `json:"edited_timestamp"`
	TSS                  bool                 `json:"tts"`
	MentionEveryone      bool                 `json:"mention_everyone"`
	Mentions             []User               `json:"mentions"`
	MentionRoles         []string             `json:"mention_roles"`
	MentionChannels      []ChannelMention     `json:"mention_chnnels"`
	Attachments          []Attachment         `json:"attachments"`
	Embeds               []Embed              `json:"embeds"`
	Reactions            []Reaction           `json:"reactions"`
	Nonce                json.RawMessage      `json:"nonce,omitempty"`
	Pinned               bool                 `json:"pinned,omitempty"`
	WebhookId            string               `json:"webhook_id,omitempty"`
	Type                 int                  `json:"type,omitempty"`
	Activity             MessageActivity      `json:"activity,omitempty"`
	Application          Application          `json:"application,omitempty"`
	ApplicationId        string               `json:"application_id,omitempty"`
	MessageReference     MessageReference     `json:"message_reference,omitempty"`
	Flags                int                  `json:"flags,omitempty"`
	ReferencedMessage    *Message             `json:"referenced_message,omitempty"`
	Interaction          MessageInteraction   `json:"interaction,omitempty"`
	Thread               Channel              `json:"thread,omitempty"`
	Components           []json.RawMessage    `json:"components,omitempty"`
	StickerItems         []StickerItem        `json:"sticker_items,omitempty"`
	Stickers             []Sticker            `json:"stickers,omitempty"`
	Position             int                  `json:"position,omitempty"`
	RoleSubscriptionData RoleSubscriptionData `json:"role_subscription_data,omitempty"`
	Resolved             ResolvedData         `json:"resolved,omitempty"`
}

type ChannelMention struct {
	Id      string `json:"id"`
	GuildId string `json:"guild_id"`
	Type    int    `json:"type"`
	Name    string `json:"name"`
}

type Reaction struct {
	Count        int                  `json:"count"`
	CountDetails ReactionCountDetails `json:"count_details"`
	Me           bool                 `json:"me"`
	MeBurst      bool                 `json:"me_burst"`
	Emoji        Emoji                `json:"emoji"`
	BurstColors  []string             `json:"burst_colors"`
}

type ReactionCountDetails struct {
	Burst  int `json:"burst"`
	Normal int `json:"normal"`
}

type Emoji struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	Roles          []string `json:"roles"`
	User           []User   `json:"user"`
	RequiredColons bool     `json:"require_colons,omitempty"`
	Managed        bool     `json:"managed,omitempty"`
	Animated       bool     `json:"animated,omitempty"`
	Available      bool     `json:"available,omitempty"`
}

type MessageActivity struct {
	Type    int    `json:"type"`
	PartyId string `json:"party_id"`
}

type Application struct {
	Id                             string        `json:"id"`
	Name                           string        `json:"name"`
	Icon                           string        `json:"icon"`
	Description                    string        `json:"description"`
	RPCOrigins                     []string      `json:"rpc_origins,omitempty"`
	BotPublic                      bool          `json:"bot_public"`
	BotRequireCodeGrant            bool          `json:"bot_require_code_grant"`
	Bot                            User          `json:"bot,omitempty"`
	TermsOfServiceUrl              string        `json:"terms_of_service_url,omitempty"`
	PrivacyPolicyUrl               string        `json:"privacy_policy_url,omitempty"`
	Owner                          User          `json:"owner,omitempty"`
	Summary                        string        `json:"summary"`
	VerifyKey                      string        `json:"verify_key"`
	Team                           Team          `json:"team"`
	GuildId                        string        `json:"guild_id"`
	Guild                          Guild         `json:"guild"`
	PrimarySkuId                   string        `json:"primary_sku_id,omitempty"`
	Slug                           string        `json:"slug,omitempty"`
	CoverImage                     string        `json:"cover_image,omitempty"`
	Flags                          int           `json:"flags,omitempty"`
	ApproximateGuildCount          int           `json:"approximate_guild_count,omitempty"`
	RedirectUris                   []string      `json:"redirect_uris,omitempty"`
	InteractionsEndpointUrl        string        `json:"interactions_endpoint_url,omitempty"`
	RoleConnectionsVerificationUrl string        `json:"role_connections_verification_url,omitempty"`
	Tags                           []string      `json:"tags,omitempty"`
	InstallParams                  InstallParams `json:"install_params,omitempty"`
	CustomInstallUrl               string        `json:"custom_install_url,omitempty"`
}

type Team struct {
	Icon        string       `json:"icon"`
	Id          string       `json:"id"`
	Members     []TeamMember `json:"members"`
	Name        string       `json:"name"`
	OwnerUserId string       `json:"owner_user_id"`
}

type TeamMember struct {
	MembershipState int    `json:"membership_state"`
	TeamId          string `json:"team_id"`
	User            User   `json:"user"`
	Role            string `json:"role"`
}

type Guild struct {
	Id                         string        `json:"id"`
	Name                       string        `json:"name"`
	Icon                       string        `json:"icon"`
	IconHash                   string        `json:"icon_hash,omitempty"`
	Splash                     string        `json:"splash"`
	DiscoverySplash            string        `json:"discovery_splash"`
	Owner                      bool          `json:"owner,omitempty"`
	OwnerId                    string        `json:"owner_id"`
	Permissions                string        `json:"permissions,omitempty"`
	Region                     string        `json:"region,omitempty"`
	AfkChannelId               string        `json:"afk_channel_id"`
	AfkTimeout                 int           `json:"afk_timeout"`
	WidgetEnabled              bool          `json:"widget_enabled,omitempty"`
	WidgetChannelId            string        `json:"widget_channel_id,omitempty"`
	VerificationLevel          int           `json:"verification_level"`
	DefaultMessageNotification int           `json:"default_message_notification"`
	ExplicitContentFilter      int           `json:"explicit_content_filter"`
	Roles                      []Roles       `json:"roles"`
	Emojis                     []Emoji       `json:"emojis"`
	Features                   []string      `json:"features"`
	MFALevel                   int           `json:"mfa_level"`
	ApplicationId              string        `json:"application_id"`
	SystemChannelFlags         int           `json:"system_channel_flags"`
	RulesChannelId             string        `json:"rules_channel_id"`
	MaxPresences               int           `json:"max_presences,omitempty"`
	MaxMembers                 int           `json:"max_members,omitempty"`
	VanityUrlCode              string        `json:"vanity_url_code"`
	Description                string        `json:"description"`
	Banner                     string        `json:"banner"`
	PremiumTier                int           `json:"premium_tier"`
	PremiumSubscriptionCount   int           `json:"premium_subscription_count,omitempty"`
	PreferredLocale            string        `json:"preferred_locale"`
	PublicUpdatesChannelId     string        `json:"public_updates_channel_id"`
	MaxVideoChannelUsers       int           `json:"max_video_channel_users,omitempty"`
	MaxStageVideoChannelUsers  int           `json:"max_stage_video_channel_users,omitempty"`
	ApproximateMemberCount     int           `json:"approximateMemberCount,omitempty"`
	ApproximatePresenceCount   int           `json:"approximate_presences_count,omitempty"`
	WelcomeScreen              WelcomeScreen `json:"welcome_screen,omitempty"`
	NSFWLevel                  int           `json:"nsfw_level"`
	Stickers                   []Sticker     `json:"sticker,omitempty"`
	PremiumProgressBarEnabled  bool          `json:"premium_progress_bar_enabled"`
	SafetyAlertsChannelId      string        `json:"safety_alerts_channel_id"`
}

type Roles struct {
	Id           string     `json:"id"`
	Name         string     `json:"name"`
	Color        int        `json:"color"`
	Hoist        bool       `json:"hoist"`
	Icon         string     `json:"icon,omitempty"`
	UnicodeEmoji string     `json:"unicode_emoji,omitempty"`
	Position     int        `json:"position"`
	Permissions  string     `json:"permissions"`
	Managed      bool       `json:"managed"`
	Mentionable  bool       `json:"mentionalbe"`
	Tags         []RoleTags `json:"tags,omitempty"`
	Flags        int        `json:"flags"`
}

type RoleTags struct {
	BotId                 string `json:"bot_id,omitempty"`
	IntegrationId         string `json:"integration_id,omitempty"`
	PremiumSubscriber     bool   `json:"premium_subscriber,omitempty"`
	SubscriptionListingId bool   `json:"subscription_listing_id,omitempty"`
	AvailableForPurchase  bool   `json:"available_for_purchase,omitempty"`
	GuildConnections      bool   `json:"guild_connections,omitempty"`
}

type WelcomeScreen struct {
	Description     string                 `json:"description"`
	WelcomeChannels []WelcomeScreenChannel `json:"welcome_channel"`
}

type WelcomeScreenChannel struct {
	ChannelId   string `json:"channel_id"`
	Description string `json:"description"`
	EmojiId     string `json:"emoji_id"`
	Emojiname   string `json:"emoji_name"`
}

type Sticker struct {
	Id          string `json:"id"`
	PackId      string `json:"pack_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	Asset       string `json:"asset,omitempty"`
	Type        int    `json:"type"`
	FormatType  int    `json:"format_type"`
	Available   bool   `json:"available,omitempty"`
	GuildId     string `json:"guild_id,omitempty"`
	User        User   `json:"user,omitempty"`
	SortValue   int    `json:"sort_value,omitempty"`
}

type InstallParams struct {
	Scopes      []string `json:"scopes"`
	Permissions string   `json:"permissions"`
}

type MessageReference struct {
	MessageId       string `json:"message_id"`
	ChannelId       string `json:"channel_id"`
	GuildID         string `json:"guild_id"`
	FailIfNotExists bool   `json:"fail_if_not_exists"`
}

type MessageInteraction struct {
	Id     string      `json:"id"`
	Type   int         `json:"type"`
	Name   string      `json:"name"`
	User   User        `json:"user"`
	Member GuildMember `json:"member,omitempty"`
}

type StickerItem struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	FormatType int    `json:"format_type"`
}

type RoleSubscriptionData struct {
	RoleSubscriptionListingId string `json:"role_subscription_listing_id"`
	TierName                  string `json:"tier_name"`
	TotalMonthsSubscribed     int    `json:"total_months_subscribed"`
	IsRenewal                 bool   `json:"is_renewal"`
}
