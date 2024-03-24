package voice

import "encoding/json"

/*
GatewayEventType

GatewayEventTypeDispatch An event was dispatched.

GatewayEventTypeHeartbeat Fired periodically by the client to keep the connection alive.

GatewayEventTypeIdentify Starts a new session during the initial handshake.

GatewayEventTypePresenceUpdate Update the client's presence.

GatewayEventTypeVoiceStateUpdate Used to join/leave or move between voice channels.

GatewayEventTypeResume Resume a previous session that was disconnected.

GatewayEventTypeReconnect You should attempt to reconnect and resume immediately.

GatewayEventTypeRequestGuildMembers Request information about offline guild members in a large guild.

GatewayEventTypeInvalidSession The session has been invalidated. You should reconnect and identify/resume accordingly.

GatewayEventTypeHello Sent immediately after connecting, contains the heartbeat_interval to use.

GatewayEventTypeHeartbeatACK Sent in response to receiving a heartbeat to acknowledge that it has been received.

GatewayEventTypeRequestSoundboardSounds Request information about soundboard sounds in a set of guilds.
*/
type GatewayEventType int

const (
	GatewayEventTypeIdentify           GatewayEventType = 0
	GatewayEventTypeSelectProtocol     GatewayEventType = 1
	GatewayEventTypeReady              GatewayEventType = 2
	GatewayEventTypeHeartbeat          GatewayEventType = 3
	GatewayEventTypeSessionDescription GatewayEventType = 4
	GatewayEventTypeSpeaking           GatewayEventType = 5
	GatewayEventTypeHeartbeatAck       GatewayEventType = 6
	GatewayEventTypeResume             GatewayEventType = 7
	GatewayEventTypeHello              GatewayEventType = 8
	GatewayEventTypeResumed            GatewayEventType = 9
	GatewayEventTypeClientsConnect     GatewayEventType = 11
	GatewayEventTypeClientDisconnect   GatewayEventType = 13
	// TODO: DAVE
)

type GatewayEvent struct {
	OP GatewayEventType `json:"op"`
	D  json.RawMessage  `json:"d"`
}

type IdentifyEventData struct {
	ServerId               string `json:"server_id"`
	UserId                 string `json:"user_id"`
	SessionId              string `json:"session_id"`
	Token                  string `json:"token"`
	MaxDaveProtocolVersion int    `json:"max_dave_protocol_version"`
}

type SelectProtocolEventData struct {
	Protocol string                      `json:"protocol"`
	Data     SelectProtocolEventDataData `json:"data"`
}

type SelectProtocolEventDataData struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Mode    string `json:"mode"`
}

type ReadyEventData struct {
	SSRC  uint32   `json:"ssrc"`
	IP    string   `json:"ip"`
	Port  int      `json:"port"`
	Modes []string `json:"modes"`
}

type HeartbeatEventData struct {
	T      int64 `json:"t"`
	SeqAck int64 `json:"seq_ack"`
}

type SessionDescriptionEventData struct {
	Mode                string `json:"mode"`
	SecretKey           []byte `json:"secret_key"`
	DaveProtocolVersion int    `json:"dave_protocol_version"`
}
type SpeakingFlag uint8

const (
	SpeakingMicrophone SpeakingFlag = 1 << iota // 1
	SpeakingSoundshare                          // 2
	SpeakingPriority                            // 4
)

type SpeakingEventData struct {
	Speaking bool   `json:"speaking"`
	Delay    int    `json:"delay"`
	SSRC     uint32 `json:"ssrc"`
}

type HelloEventData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}
