package message

type Name string

const (
	NameInitivexpand2    Name = "initivexpand2"
	NameInitserver       Name = "initserver"
	NameClientInitIV     Name = "clientinitiv"
	NameClientek         Name = "clientek"
	NameClientDisconnect Name = "clientdisconnect"
)

type ClientDisconnect struct {
	ReasonId  int    `teamspeak:"reasonid"`
	ReasonMsg string `teamspeak:"reasonmsg"`
}

func NewClientDisconnect(reason string) *ClientDisconnect {
	return &ClientDisconnect{
		ReasonId:  8,
		ReasonMsg: reason,
	}
}
