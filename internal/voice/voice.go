package voice

//
//import (
//	"Jabba_The_Bot/internal/ws"
//	"encoding/json"
//	"fmt"
//	"log"
//	"net"
//	"sync"
//	"time"
//)
//
//type Connection struct {
//	GuildId   string
//	ChannelId string
//	UserId    string
//	Token     string
//	Endpoint  string
//	SessionId string
//
//	WSClient      *ws.Client
//	UDPConnection *net.UDPConn
//
//	SSRC int
//	Port int
//	IP   string
//
//	Modes     []string
//	SecretKey [32]byte
//
//	sequence  int64
//	timestamp uint32
//
//	waitGroup sync.WaitGroup
//}
//
//func NewConnection(guildId, channelId, userId, token, endpoint, sessionId string) *Connection {
//	return &Connection{
//		GuildId:   guildId,
//		ChannelId: channelId,
//		UserId:    userId,
//		Token:     token,
//		Endpoint:  endpoint,
//		SessionId: sessionId,
//		WSClient:  ws.NewClient(),
//		sequence:  -1,
//	}
//}
//
//func (c *Connection) Connect() error {
//	url := fmt.Sprintf("wss://%s/?v=8", c.Endpoint)
//	log.Printf("Connecting to Voice Gateway: %s\n", url)
//
//	err := c.WSClient.Connect(url)
//	if err != nil {
//		return err
//	}
//
//	c.Identify()
//
//	go c.listen()
//
//	return nil
//}
//
//func (c *Connection) Identify() {
//	payload := map[string]interface{}{
//		"server_id":  c.GuildId,
//		"user_id":    c.UserId,
//		"session_id": c.SessionId,
//		"token":      c.Token,
//	}
//
//	c.WSClient.Send(0, payload)
//}
//
//func (c *Connection) Send(op int, data interface{}) {
//	c.WSClient.Send(op, data)
//}
//
//func (c *Connection) listen() {
//	for {
//		var msg ws.Message
//		err := c.WSClient.Conn.ReadJSON(&msg)
//		if err != nil {
//			log.Println("Voice Gateway read error:", err)
//			return
//		}
//
//		if msg.S != nil {
//			c.sequence = *msg.S
//		}
//
//		switch msg.OP {
//		case 2: // Ready
//			var data struct {
//				SSRC  int      `json:"ssrc"`
//				IP    string   `json:"ip"`
//				Port  int      `json:"port"`
//				Modes []string `json:"modes"`
//			}
//			json.Unmarshal(msg.D, &data)
//			c.SSRC = data.SSRC
//			c.IP = data.IP
//			c.Port = data.Port
//			c.Modes = data.Modes
//
//			log.Printf("Voice Ready: SSRC=%d, IP=%s, Port=%d, Modes=%v\n", c.SSRC, c.IP, c.Port, c.Modes)
//			// Next: IP Discovery and Select Protocol
//			go c.discoverIP()
//
//		case 4: // Session Description
//			var data struct {
//				SecretKey [32]byte `json:"secret_key"`
//				Mode      string   `json:"mode"`
//			}
//			json.Unmarshal(msg.D, &data)
//			c.SecretKey = data.SecretKey
//			log.Println("Voice Session Description received, encryption key ready")
//
//		case 6:
//			log.Println("Heartbeat ACK received")
//
//		case 8: // Hello
//			var data struct {
//				HeartbeatInterval float64 `json:"heartbeat_interval"`
//			}
//			json.Unmarshal(msg.D, &data)
//
//			c.WSClient.StartHeartbeat(int(data.HeartbeatInterval), c.heartbeatFunc)
//
//		case 9: // Invalid Session
//			log.Println("Voice Session invalidated (Op 9). Reconnecting...")
//			c.Identify()
//		}
//
//	}
//}
//
//func (c *Connection) heartbeatFunc() error {
//	payload := map[string]interface{}{
//		"t":       time.Now().Unix() / int64(time.Millisecond),
//		"seq_ack": c.sequence,
//	}
//	return c.WSClient.Send(3, payload)
//}
//
//func (c *Connection) discoverIP() {
//	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.IP, c.Port))
//	if err != nil {
//		log.Println("Error resolving UDP address:", err)
//		return
//	}
//
//	conn, err := net.DialUDP("udp", nil, addr)
//	if err != nil {
//		log.Println("Error dialing UDP:", err)
//		return
//	}
//	c.UDPConnection = conn
//
//	// IP Discovery packet (74 bytes)
//	packet := make([]byte, 74)
//	// Type (1-2)
//	packet[0] = 0x0
//	packet[1] = 0x1
//	// Length (2-4)
//	packet[2] = 0x0
//	packet[3] = 0x46
//	// SSRC (4-8)
//	packet[4] = byte(c.SSRC >> 24)
//	packet[5] = byte(c.SSRC >> 16)
//	packet[6] = byte(c.SSRC >> 8)
//	packet[7] = byte(c.SSRC)
//
//	_, err = conn.Write(packet)
//	if err != nil {
//		log.Println("Error writing to UDP:", err)
//		return
//	}
//
//	res := make([]byte, 74)
//	_, err = conn.Read(res)
//	if err != nil {
//		log.Println("Error reading from UDP:", err)
//		return
//	}
//
//	// Extract IP and Port from response
//	externalIP := string(res[8:72])
//	// find null terminator
//	for i, b := range externalIP {
//		if b == 0 {
//			externalIP = externalIP[:i]
//			break
//		}
//	}
//	externalPort := uint16(res[72])<<8 | uint16(res[73])
//
//	log.Printf("External IP: %s, Port: %d\n", externalIP, externalPort)
//
//	c.SelectProtocol(externalIP, externalPort)
//}
//
//func (c *Connection) SelectProtocol(ip string, port uint16) {
//	mode := "aead_xchacha20_poly1305_rtpsize"
//	found := false
//	for _, m := range c.Modes {
//		if m == mode {
//			found = true
//			break
//		}
//	}
//
//	if !found && len(c.Modes) > 0 {
//		mode = c.Modes[0]
//	}
//
//	log.Printf("Selecting protocol with mode: %s\n", mode)
//
//	payload := map[string]interface{}{
//		"protocol": "udp",
//		"data": map[string]interface{}{
//			"address": ip,
//			"port":    port,
//			"mode":    mode,
//		},
//	}
//	c.Send(1, payload)
//}
