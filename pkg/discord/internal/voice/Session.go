package voice

import (
	"Jabba_The_Bot/pkg/discord"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type State int

const (
	StateCreated State = iota
	StateRequestingToJoin
	StateConnectingToWebsocket
	StateSocketOpen    // Connected to Voice WS
	StateDiscoveringIP // Doing UDP magic
	StateReady         // Got Secret Key, ready to send audio
	StateError
)

type Session struct {
	guildId string
	userId  string

	sessionId string
	token     string
	endpoint  string

	secretKey      []byte
	encryptionMode string
	sequence       uint16
	timestamp      uint32
	nonce          uint32

	lastSeqAck int64

	udpConn *net.UDPConn
	wsMutex sync.Mutex
	wsConn  *websocket.Conn

	AudioInput chan []byte

	StateUpdates chan State
	isReady      bool

	isSpeaking bool
	ssrc       uint32
	gcm        cipher.AEAD

	perf *Perf
}

func NewSession(guildId string, userId string) *Session {
	session := &Session{
		guildId: guildId,
		userId:  userId,

		AudioInput:   make(chan []byte),
		StateUpdates: make(chan State, 10),
		isReady:      false,

		lastSeqAck: -1,
	}

	session.StateUpdates <- StateCreated

	return session
}

func (s *Session) writeJson(v interface{}) error {
	s.wsMutex.Lock()
	defer s.wsMutex.Unlock()
	return s.wsConn.WriteJSON(v)
}

func (s *Session) SetSpeaking(isSpeaking bool) {
	s.isSpeaking = isSpeaking

	speakingEventData := SpeakingEventData{
		Speaking: isSpeaking,
		Delay:    0,
		SSRC:     s.ssrc,
	}

	speakingEventPayload, _ := json.Marshal(speakingEventData)

	speakingEvent := GatewayEvent{
		OP: GatewayEventTypeSpeaking,
		D:  speakingEventPayload,
	}

	err := s.writeJson(speakingEvent)
	if err != nil {
		log.Println("Error sending speaking event:", err)
	}
}

func (s *Session) SetSessionId(SessionId string) {
	s.sessionId = SessionId
	slog.Debug("Set Session ID", "SessionId", SessionId)

	s.initWebsocketConnectionIfReady()
}

func (s *Session) GetSessionId() string {
	return s.sessionId
}

func (s *Session) SetToken(Token string) {
	s.token = Token
	slog.Debug("Set Token", "Token", Token)

	s.initWebsocketConnectionIfReady()
}

func (s *Session) SetEndpoint(Endpoint string) {
	s.endpoint = Endpoint
	slog.Debug("Set Endpoint", "Endpoint", Endpoint)

	s.initWebsocketConnectionIfReady()
}

func (s *Session) initWebsocketConnectionIfReady() {
	if (s.sessionId == "") || (s.token == "") || (s.endpoint == "") {
		return
	}

	s.wsMutex.Lock()
	if s.wsConn != nil {
		s.wsMutex.Unlock()
		return
	}
	s.wsMutex.Unlock()

	s.StateUpdates <- StateConnectingToWebsocket

	url := fmt.Sprintf("wss://%s?v=8", s.endpoint)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}
	s.wsConn = conn

	go s.listen()

	s.identify()
}

func (s *Session) listen() {
	defer s.wsConn.Close()

	for {
		messageType, eventBytes, err := s.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, 4014, 4006, 4009) {
				slog.Error("Discord closed connection", "Error", err)
			} else {
				slog.Error("Websocket read error (Stopping Listen)", "Error", err)
			}
			return
		}

		if messageType == websocket.BinaryMessage {
			if len(eventBytes) < 3 {
				slog.Debug("Received truncated binary message:", "Message", eventBytes)
				continue
			}

			seqNum := binary.BigEndian.Uint16(eventBytes[:2])
			s.lastSeqAck = int64(seqNum)

			opcode := eventBytes[2]
			payload := eventBytes[3:]

			slog.Debug("Received binary voice Event:", "Seq", seqNum, "Opcode", opcode, "PayloadLen", len(payload))

			switch opcode {
			case 0x19:
				slog.Info("Discord sent DAVE External Sender Package (Op 25)")
			case 0x1B:
				slog.Info("Discord sent DAVE MLS Proposal (Op 27)")
			default:
				slog.Debug("Discord sent unhabdled Binary Code:", "Opcode", opcode)
			}

			continue
		}

		var event GatewayEvent
		if err := json.Unmarshal(eventBytes, &event); err != nil {
			slog.Error("Error parsing Gateway Event:", "Error", err)
			continue
		}

		slog.Debug("Received Voice Event", "Event", event)

		switch event.OP {
		case GatewayEventTypeReady:
			var readyEventData ReadyEventData
			if err := json.Unmarshal(event.D, &readyEventData); err != nil {
				slog.Error("Error parsing Ready Event Data:", "Error", err)
				continue
			}

			mode := "aead_aes256_gcm_rtpsize"
			found := false
			for _, m := range readyEventData.Modes {
				if m == mode {
					found = true
					break
				}
			}

			if !found && len(readyEventData.Modes) > 0 {
				mode = readyEventData.Modes[0]
			}

			s.ssrc = readyEventData.SSRC
			ip := readyEventData.IP
			port := readyEventData.Port

			log.Printf("Voice Ready: SSRC=%d, IP=%s, Port=%d, Modes=%v\n", s.ssrc, ip, port, mode)

			addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
			if err != nil {
				log.Println("Error resolving UDP address:", err)
				break
			}

			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				log.Println("Error dialing UDP:", err)
				break
			}
			s.udpConn = conn

			// IP Discovery packet (74 bytes)
			packet := make([]byte, 74)
			// Type (1-2)
			packet[0] = 0x0
			packet[1] = 0x1
			// Length (2-4)
			packet[2] = 0x0
			packet[3] = 0x46
			// SSRC (4-8)
			packet[4] = byte(s.ssrc >> 24)
			packet[5] = byte(s.ssrc >> 16)
			packet[6] = byte(s.ssrc >> 8)
			packet[7] = byte(s.ssrc)

			_, err = conn.Write(packet)
			if err != nil {
				log.Println("Error writing to UDP:", err)
				break
			}

			res := make([]byte, 74)
			_, err = conn.Read(res)
			if err != nil {
				log.Println("Error reading from UDP:", err)
				break
			}

			externalIP := string(res[8:72])
			for i, b := range externalIP {
				if b == 0 {
					externalIP = externalIP[:i]
					break
				}
			}
			externalPort := uint16(res[72])<<8 | uint16(res[73])

			log.Printf("External IP: %s, Port: %d\n", externalIP, externalPort)

			selectProtocolEventData := SelectProtocolEventData{
				Protocol: "udp",
				Data: SelectProtocolEventDataData{
					Address: externalIP,
					Port:    int(externalPort),
					Mode:    mode,
				},
			}

			selectProtocolEventPayload, err := json.Marshal(selectProtocolEventData)
			if err != nil {
				log.Println("Error marshalling Select Protocol Event Data:", err)
				break
			}

			gatewayEvent := GatewayEvent{
				OP: GatewayEventTypeSelectProtocol,
				D:  selectProtocolEventPayload,
			}

			err = s.writeJson(gatewayEvent)
			if err != nil {
				slog.Error("Error writing Gateway Event:", "Error", err)
				return
			}

		case GatewayEventTypeSessionDescription:
			var data SessionDescriptionEventData
			err := json.Unmarshal(event.D, &data)
			if err != nil {
				slog.Error("Error parsing Session Description Event Data:", "Error", err)
				return
			}

			s.secretKey = data.SecretKey
			s.encryptionMode = data.Mode

			block, err := aes.NewCipher(s.secretKey)
			if err != nil {
				slog.Error("Error creating AES cipher:", "Error", err)
				return
			}

			gcm, err := cipher.NewGCM(block)
			if err != nil {
				slog.Error("Error creating GCM:", "Error", err)
				return
			}

			s.gcm = gcm

			s.isReady = true
			s.StateUpdates <- StateReady

		case GatewayEventTypeHeartbeatAck:
			slog.Debug("Received Voice Heartbeat ACK")

		case GatewayEventTypeHello:
			var helloData HelloEventData
			if err := json.Unmarshal(event.D, &helloData); err != nil {
				slog.Error("Error parsing Hello Event Data:", "Error", err)
				continue
			}

			go s.startHeartbeat(helloData.HeartbeatInterval)

		default:
			slog.Debug("Received unhandled Voice Event:", "Event", event.OP)
		}
	}
}

func (s *Session) identify() {
	identifyEventData := IdentifyEventData{
		ServerId:               s.guildId,
		UserId:                 s.userId,
		SessionId:              s.sessionId,
		Token:                  s.token,
		MaxDaveProtocolVersion: 0,
	}

	identifyEventPayload, _ := json.Marshal(identifyEventData)

	gatewayEvent := GatewayEvent{
		OP: GatewayEventTypeIdentify,
		D:  identifyEventPayload,
	}

	err := s.writeJson(gatewayEvent)
	if err != nil {
		return
	}
}

func (s *Session) startHeartbeat(interval int) {
	slog.Debug("Starting heartbeat", "interval", interval)

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			slog.Debug("Sending Voice Websocket heartbeat")

			heartbeatEventData := HeartbeatEventData{
				T:      time.Now().Unix(),
				SeqAck: s.lastSeqAck,
			}

			heartbeatEventPayload, err := json.Marshal(heartbeatEventData)
			if err != nil {
				slog.Debug("Error marshalling heartbeat payload.", "Error", err)
				return
			}

			heartbeatEvent := GatewayEvent{
				OP: GatewayEventTypeHeartbeat,
				D:  heartbeatEventPayload,
			}

			if err := s.writeJson(heartbeatEvent); err != nil {
				slog.Debug("Error sending heartbeat.", "Error", err)
			}
		}
	}
}

func (s *Session) WaitReady() {
	if s.isReady {
		return
	}

	for {
		select {
		case state := <-s.StateUpdates:
			if state == StateReady {
				return
			}
		}
	}
}

func (s *Session) SendFrame(opusFrame []byte) error {

	if s.udpConn == nil {
		return fmt.Errorf("voice udp not ready")
	}
	if s.gcm == nil {
		return fmt.Errorf("voice encryption not ready")
	}

	header := make([]byte, 12)
	nonceBytes := make([]byte, 12)

	header[0] = 0x80
	header[1] = 0x78
	binary.BigEndian.PutUint32(header[8:], s.ssrc)

	if !s.isSpeaking {
		s.SetSpeaking(true)
	}

	if len(opusFrame) == 0 {
		return fmt.Errorf("Opus Frame cannot be empty")
	}

	binary.BigEndian.PutUint16(header[2:], s.sequence)
	binary.BigEndian.PutUint32(header[4:], s.timestamp)

	binary.LittleEndian.PutUint32(nonceBytes[:4], s.nonce)
	s.nonce++

	encrypted := s.gcm.Seal(nil, nonceBytes, opusFrame, header)
	encrypted = append(encrypted, nonceBytes[:4]...)
	encrypted = append(header, encrypted...)

	_, err := s.udpConn.Write(encrypted)
	if err != nil {
		return err
	}

	s.sequence++
	s.timestamp += uint32(960)

	return nil
}

func (s *Session) Play(source discord.AudioSource) error {
	if s.udpConn == nil {
		return fmt.Errorf("voice udp not ready")
	}
	if s.gcm == nil {
		return fmt.Errorf("voice encryption not ready")
	}

	if s.perf == nil {
		s.perf = &Perf{}
		StartPerfLogger(s.perf, 5*time.Second)
	}
	p := s.perf

	header := make([]byte, 12)
	nonceBytes := make([]byte, 12)

	// construct constant header part
	header[0] = 0x80
	header[1] = 0x78
	binary.BigEndian.PutUint32(header[8:], s.ssrc)

	s.SetSpeaking(true)
	defer s.SetSpeaking(false)

	const frameDur = 20 * time.Millisecond

	// Drift-free pacing: maintain an absolute schedule.
	// Start "next" as now + frameDur so the first send happens after one frame period.
	next := time.Now().Add(frameDur)

	//ticker := time.NewTicker(20 * time.Millisecond)
	//defer ticker.Stop()

	for {
		frameStart := time.Now()

		t0 := time.Now()
		opusFrame, err := source.ProvideFrame()
		p.Demux.Add(time.Since(t0)) // rename to "provide" if you want

		if err == io.EOF {
			slog.Debug("Audio Source: EOF")
			return nil
		}
		if err != nil {
			return fmt.Errorf("audio source: %w", err)
		}
		if len(opusFrame) == 0 {
			continue
		}

		// ---- packet/crypto stage (track as "encode" for now)
		t0 = time.Now()

		binary.BigEndian.PutUint16(header[2:], s.sequence)
		binary.BigEndian.PutUint32(header[4:], s.timestamp)

		binary.LittleEndian.PutUint32(nonceBytes[:4], s.nonce)
		s.nonce++

		encrypted := s.gcm.Seal(nil, nonceBytes, opusFrame, header)
		encrypted = append(encrypted, nonceBytes[:4]...)
		encrypted = append(header, encrypted...)

		p.encode.Add(time.Since(t0))

		t0 = time.Now()
		if t0.Before(next) {
			time.Sleep(time.Until(next))
			t0 = time.Now()
		}

		if t0.After(next) {
			p.Behind.Add(t0.Sub(next))
		}

		t0 = time.Now()
		_, err = s.udpConn.Write(encrypted)
		p.Send.Add(time.Since(t0))

		if err != nil {
			slog.Warn("UDP send error", "err", err)
		}

		s.sequence++
		s.timestamp += uint32(960)

		p.Total.Add(time.Since(frameStart))

		next = next.Add(frameDur)

		for time.Since(next) > 200*time.Millisecond {
			next = next.Add(frameDur)
		}
	}
}

func (s *Session) PlayTest(source discord.AudioSource) error {
	if s.udpConn == nil {
		return fmt.Errorf("voice udp not ready")
	}
	if s.gcm == nil {
		return fmt.Errorf("voice encryption not ready")
	}

	if s.perf == nil {
		s.perf = &Perf{}
		StartPerfLogger(s.perf, 5*time.Second)
	}
	p := s.perf

	header := make([]byte, 12)
	nonceBytes := make([]byte, 12)
	header[0] = 0x80
	header[1] = 0x78
	binary.BigEndian.PutUint32(header[8:], s.ssrc)

	s.SetSpeaking(true)
	defer s.SetSpeaking(false)

	const frameDur = 20 * time.Millisecond
	const busyWaitThreshold = 1 * time.Millisecond // Busy-wait the last 1ms

	next := time.Now().Add(frameDur)
	var encrypted []byte

	for {
		frameStart := time.Now()

		// ----- Send the previous packet at scheduled time -----
		if encrypted != nil {
			now := time.Now()
			timeUntilSend := next.Sub(now)

			// Sleep until we're within 1ms of the deadline
			if timeUntilSend > busyWaitThreshold {
				time.Sleep(timeUntilSend - busyWaitThreshold)
			}

			// Busy-wait for the last millisecond for precision
			for time.Now().Before(next) {
				// Tight loop - burns CPU but guarantees timing
			}

			// Track lateness
			now = time.Now()
			if now.After(next) {
				p.Behind.Add(now.Sub(next))
			}

			// Send immediately
			t0 := time.Now()
			_, err := s.udpConn.Write(encrypted)
			p.Send.Add(time.Since(t0))

			if err != nil {
				slog.Warn("UDP send error", "err", err)
			}

			s.sequence++
			s.timestamp += uint32(960)
			next = next.Add(frameDur)

			// Catchup logic
			for time.Since(next) > 200*time.Millisecond {
				next = next.Add(frameDur)
			}
		}

		// ----- Prepare next packet -----
		t0 := time.Now()
		opusFrame, err := source.ProvideFrame()
		p.Demux.Add(time.Since(t0))

		if err == io.EOF {
			slog.Debug("Audio Source: EOF")
			return nil
		}
		if err != nil {
			return fmt.Errorf("audio source: %w", err)
		}
		if len(opusFrame) == 0 {
			continue
		}

		t0 = time.Now()
		binary.BigEndian.PutUint16(header[2:], s.sequence)
		binary.BigEndian.PutUint32(header[4:], s.timestamp)
		binary.LittleEndian.PutUint32(nonceBytes[:4], s.nonce)
		s.nonce++

		encrypted = s.gcm.Seal(encrypted[:0], nonceBytes, opusFrame, header)
		encrypted = append(encrypted, nonceBytes[:4]...)
		encrypted = append(header, encrypted...)

		p.encode.Add(time.Since(t0))
		p.Total.Add(time.Since(frameStart))
	}
}

type FrameTimings struct {
	Demux  time.Duration
	Decode time.Duration
	Encode time.Duration
	Send   time.Duration
	Pacing time.Duration // time spent waiting/sleeping
	Total  time.Duration
	Behind time.Duration // how far past deadline we were (if any)
}
type stageStats struct {
	count uint64
	sum   int64
	min   int64
	max   int64
}

func (s *stageStats) Add(d time.Duration) {
	ns := int64(d)
	atomic.AddUint64(&s.count, 1)
	atomic.AddInt64(&s.sum, ns)

	// min
	for {
		old := atomic.LoadInt64(&s.min)
		if old != 0 && ns >= old {
			break
		}
		if atomic.CompareAndSwapInt64(&s.min, old, ns) {
			break
		}
	}
	// max
	for {
		old := atomic.LoadInt64(&s.max)
		if ns <= old {
			break
		}
		if atomic.CompareAndSwapInt64(&s.max, old, ns) {
			break
		}
	}
}

func (s *stageStats) snapshotAndReset() (count uint64, avg, min, max time.Duration) {
	count = atomic.SwapUint64(&s.count, 0)
	sum := atomic.SwapInt64(&s.sum, 0)
	minNs := atomic.SwapInt64(&s.min, 0)
	maxNs := atomic.SwapInt64(&s.max, 0)
	if count == 0 {
		return 0, 0, 0, 0
	}
	return count,
		time.Duration(sum / int64(count)),
		time.Duration(minNs),
		time.Duration(maxNs)
}

type Perf struct {
	Demux, decode, encode, Send, pacing, Behind, Total stageStats
}

func StartPerfLogger(p *Perf, every time.Duration) {
	t := time.NewTicker(every)
	go func() {
		for range t.C {
			logStage := func(name string, s *stageStats) {
				n, avg, min, max := s.snapshotAndReset()
				if n == 0 {
					return
				}
				log.Printf("%s: n=%d avg=%s min=%s max=%s", name, n, avg, min, max)
			}
			logStage("demux", &p.Demux)
			logStage("decode", &p.decode)
			logStage("encode", &p.encode)
			logStage("send", &p.Send)
			logStage("pacing", &p.pacing)
			logStage("behind", &p.Behind)
			logStage("total", &p.Total)
		}
	}()
}
