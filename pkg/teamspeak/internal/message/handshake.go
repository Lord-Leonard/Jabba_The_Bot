package message

import (
	"fmt"
)

type ClientInitIV struct {
	AlphaB64 string
	OmegaB64 string
	OT       int
	IP       string
}

func NewClientInitIV(alpha, omega string, ot int, ip string) ClientInitIV {
	return ClientInitIV{
		AlphaB64: alpha,
		OmegaB64: omega,
		OT:       ot,
		IP:       ip,
	}
}

func (c ClientInitIV) Encode() []byte {
	return []byte(fmt.Sprintf(
		"%s alpha=%s omega=%s ot=%d ip=%s",
		NameClientInitIV,
		c.AlphaB64,
		c.OmegaB64,
		c.OT,
		c.IP,
	))
}

type Initivexpand2Payload struct {
	L     string `teamspeak:"l"`
	Beta  string `teamspeak:"beta"`
	Omega string `teamspeak:"omega"`
	Ot    string `teamspeak:"ot"`
	Proof string `teamspeak:"proof"`
	Tvd   string `teamspeak:"tvd"`
}

type Clientek struct {
	Ek    []byte `teamspeak:"ek"`
	Proof []byte `teamspeak:"proof"`
}

type ClientInit struct {
	Nickname               string `teamspeak:"client_nickname"`
	Version                string `teamspeak:"client_version"`
	Platform               string `teamspeak:"client_platform"`
	InputHardware          bool   `teamspeak:"client_input_hardware"`
	OutputHardware         bool   `teamspeak:"client_output_hardware"`
	DefaultChannel         string `teamspeak:"client_default_channel"`
	DefaultChannelPassword string `teamspeak:"client_default_channel_password"`
	ServerPassword         string `teamspeak:"client_server_password"`
	MetaData               string `teamspeak:"client_meta_data"`
	VersionSign            string `teamspeak:"client_version_sign"`
	KeyOffset              uint64 `teamspeak:"client_key_offset"`
	NicknamePhonetic       string `teamspeak:"client_nickname_phonetic"`
	DefaultToken           string `teamspeak:"client_default_token"`
	HardwareId             string `teamspeak:"hwid"`
}

func NewClientInit(nickname, version, platform, versionSign, hardwareId string, keyOffset uint64) ClientInit {
	return ClientInit{
		Nickname:               nickname,
		Version:                version,
		Platform:               platform,
		InputHardware:          true,
		OutputHardware:         true,
		DefaultChannel:         "",
		DefaultChannelPassword: "",
		ServerPassword:         "",
		MetaData:               "",
		VersionSign:            versionSign,
		KeyOffset:              keyOffset,
		NicknamePhonetic:       "",
		DefaultToken:           "",
		HardwareId:             hardwareId,
	}
}

type InitServer struct {
	AclId    uint16 `teamspeak:"aclid"`
	ClId     uint16 `teamspeak:"clid"`
	ClientId uint16 `teamspeak:"client_id"`
}
