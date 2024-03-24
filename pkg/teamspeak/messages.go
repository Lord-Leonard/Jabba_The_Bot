package teamspeak

type MessageName string

const (
	MessageNameTextMessage MessageName = "notifytextmessage"
	MessageNameError       MessageName = "error"
)

type TextMessageTargetMode int

const (
	TextMessageTargetModeUnknown TextMessageTargetMode = iota // unknown
	TextMessageTargetModeClient                               // Send to specific client
	TextMessageTargetModeChannel                              // Send to current channel
	TextMessageTargetModeServer                               // Send to server chat

)

type SendTextMessage struct {
	TargetMode TextMessageTargetMode `teamspeak:"targetmode"`
	Message    string                `teamspeak:"msg"`
}

func NewSendTextMessage(message string) *SendTextMessage {
	return &SendTextMessage{
		TargetMode: TextMessageTargetModeChannel,
		Message:    message,
	}
}

type TextMessage struct {
	TargetMode  TextMessageTargetMode `teamspeak:"targetmode"`
	Message     string                `teamspeak:"msg"`
	Target      string                `teamspeak:"target"`
	InvokerID   string                `teamspeak:"invokerid"`
	InvokerName string                `teamspeak:"invokername"`
	InvokerUID  string                `teamspeak:"invokeruid"`
	Timestamp   string                `teamspeak:"timestamp"`
}

type CommandError struct {
	ID           int    `teamspeak:"id"`
	Message      string `teamspeak:"msg"`
	FailedPermID string `teamspeak:"failed_permid"`
	ReturnCode   string `teamspeak:"return_code"`
	ExtraMessage string `teamspeak:"extra_msg"`
}

type ClientUpdate struct {
	Nickname           *string `teamspeak:"client_nickname"`
	IsMuted            *bool   `teamspeak:"client_input_muted"`
	IsOutputMuted      *bool   `teamspeak:"client_output_muted"`
	IsAway             *bool   `teamspeak:"client_away"`
	AwayMessage        *string `teamspeak:"client_away_message"`
	HasInputHardware   *bool   `teamspeak:"client_input_hardware"`
	HasOutputHardware  *bool   `teamspeak:"client_output_hardware"`
	IsChannelCommander *bool   `teamspeak:"client_is_channel_commander"`
	FlagAvatar         *string `teamspeak:"client_flag_avatar"`
	PhoneticNickname   *string `teamspeak:"client_nickname_phonetic"`
	HasRequestedTalk   *bool   `teamspeak:"client_talk_request"`
	talkRequestMessage *string `teamspeak:"client_talk_request_msg"`
	IsRecording        *bool   `teamspeak:"client_is_recording"`
	Badges             *string `teamspeak:"client_badges"`
}

type ClientEdit struct {
	ClientId          uint16  `teamspeak:"clid"`
	ClientDescription *string `teamspeak:"client_description"`
	IsClientTalker    *bool   `teamspeak:"client_is_talker"`
}
