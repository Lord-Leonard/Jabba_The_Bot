package client

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/handshake"
	"Jabba_The_Bot/pkg/teamspeak/internal/message"
	"Jabba_The_Bot/pkg/teamspeak/internal/packet"
	"Jabba_The_Bot/pkg/teamspeak/internal/session"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
)

type Client struct {
	conn      *net.UDPConn
	raddr     *net.UDPAddr
	errOnce   sync.Once
	closeOnce sync.Once
	closeErr  error
	sendMu    sync.Mutex
	stateMu   sync.RWMutex
	recvCh    chan *protocol.Message
	errCh     chan error
	clientID  uint16
	voiceID   uint16
	aead      *protocol.AEAD
	config    Config
	Session   *session.Session

	cmdFragActive     bool
	cmdFragCompressed bool
	cmdFragBuf        []byte
	cmdFragNextPID    uint16
	cmdFragStarted    time.Time
}

const maxVoiceFrameBytes = 484
const closeDisconnectFlushDelay = 100 * time.Millisecond
const commandFragmentStallTimeout = 2 * time.Second

type Config struct {
	KeyPath       string
	Nickname      string
	Version       string
	VersionSign   string
	Platform      string
	HWID          string
	HashcashLevel int // TODO: Remove, see ts3protocol.md 4.1
}

func NewClient(conn *net.UDPConn, raddr *net.UDPAddr, config Config) *Client {
	return &Client{
		conn:    conn,
		raddr:   raddr,
		recvCh:  make(chan *protocol.Message, 256),
		errCh:   make(chan error, 1),
		Session: session.NewSession(nil, 0),
		config:  config,
		aead:    protocol.NewAEAD(),
	}
}

func Connect(server string, config Config) (*Client, error) {
	raddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)

	if err != nil {
		return nil, err
	}

	c := NewClient(conn, raddr, config)
	return c, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}

	c.closeOnce.Do(func() {
		// `clientdisconnect` can only be sent after handshake/session are established.
		if c.ClientID() != 0 && len(c.SharedIV()) != 0 {
			if err := c.sendClientDisconnect("Bot Stopped"); err != nil {
				slog.Debug("failed to send clientdisconnect", "error", err)
			} else {
				// Give UDP stack a brief chance to flush disconnect before closing socket.
				time.Sleep(closeDisconnectFlushDelay)
			}
		}
		c.closeErr = c.conn.Close()
	})

	return c.closeErr
}

func (c *Client) SetSharedIV(iv []byte) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	c.Session.SharedIV = append([]byte(nil), iv...)
	c.aead.SetSharedIV(c.Session.SharedIV)
}

func (c *Client) SharedIV() []byte {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return append([]byte(nil), c.Session.SharedIV...)
}

func (c *Client) SetClientID(id uint16) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.clientID = id
}

func (c *Client) ClientID() uint16 {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.clientID
}

func (c *Client) SendMessage(m any) error {
	p, err := packet.NewCommandPacket(m)
	if err != nil {
		return err
	}
	return c.send(p)
}

func (c *Client) SendVoiceFrame(codec uint8, frame []byte) error {
	if c.ClientID() == 0 || len(c.SharedIV()) == 0 {
		return errors.New("voice send not ready: handshake/session not initialized")
	}
	if len(frame) > maxVoiceFrameBytes {
		return fmt.Errorf("opus frame too large for TeamSpeak packet: %d > %d bytes", len(frame), maxVoiceFrameBytes)
	}

	p := packet.NewVoicePacket(c.nextVoiceID(), codec, frame)
	return c.send(p)
}

func (c *Client) send(p *protocol.C2SPacket) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	if p.CId == 0 {
		p.CId = c.ClientID()
	}
	var genID uint32
	if p.PId == 0 {
		pid, gid := c.Session.NextPIDAndGenID(p.Type)
		p.PId = pid
		genID = gid
	}

	if err := c.seal(p, genID); err != nil {
		return err
	}

	raw := p.Encode()

	_, err := c.conn.Write(raw)
	return err
}

func (c *Client) seal(p *protocol.C2SPacket, genID uint32) error {
	switch p.Type {
	case protocol.PTInit1:
		p.MAC = protocol.InitMac
		return nil
	case protocol.PTAckHeartbeat:
		p.MAC = protocol.SharedMacFromIV(c.SharedIV())
		return nil
	case protocol.PTCommand, protocol.PTCommandLow:
		messageName, err := protocol.ParseMessageName(string(p.Data))
		if err != nil {
			return err
		}

		if messageName == string(message.NameClientek) || messageName == string(message.NameClientInitIV) {
			return c.aead.SealBootstrap(p)
		}
		key, nonce := protocol.DeriveKeyNonce(p.Type, p.PId, genID, false, c.SharedIV())
		return c.aead.SealWithKeyNonce(p, key[:], nonce[:])
	case protocol.PTVoice, protocol.PTVoiceWhisper:
		key, nonce := protocol.DeriveKeyNonce(p.Type, p.PId, genID, false, c.SharedIV())
		return c.aead.SealWithKeyNonce(p, key[:], nonce[:])
	case protocol.PTAck, protocol.PTAckLow:
		return c.aead.SealBootstrap(p)
	default:
		return nil
	}
}

func (c *Client) open(p *protocol.S2CPacket) error {
	if p.Flags&protocol.FlagUE != 0 {
		return nil
	}

	iv := c.SharedIV()

	switch p.Type {
	case protocol.PTInit1:
		return nil
	case protocol.PTHeartbeat:
		if len(iv) == 0 {
			return nil
		}
		expected := protocol.SharedMacFromIV(iv)
		if p.MAC != expected {
			return errors.New("heartbeat MAC mismatch")
		}
		return nil
	case protocol.PTCommand, protocol.PTCommandLow:
		var sharedErr error
		if len(iv) != 0 {
			if err := c.aead.Open(p); err == nil {
				return nil
			} else {
				sharedErr = err
			}
		}
		bootErr := protocol.OpenBootstrap(p)
		if bootErr == nil {
			return nil
		}
		return fmt.Errorf("failed to decrypt command packet (shared=%v, bootstrap=%v)", sharedErr, bootErr)
	case protocol.PTAck, protocol.PTAckLow:
		var sharedErr error
		if len(iv) != 0 {
			if err := c.aead.Open(p); err == nil {
				return nil
			} else {
				sharedErr = err
			}
		}
		bootErr := protocol.OpenBootstrap(p)
		if bootErr == nil {
			return nil
		}
		return fmt.Errorf("failed to decrypt ack packet (shared=%v, bootstrap=%v)", sharedErr, bootErr)
	case protocol.PTVoice, protocol.PTVoiceWhisper:
		if len(iv) == 0 {
			return errors.New("failed to decrypt voice packet: missing shared iv")
		}
		return c.aead.Open(p)
	}

	return nil
}

func (c *Client) Initialize(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()
	go c.Listen(ctx)

	return c.initialize(ctx)
}

func (c *Client) initialize(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	A0 := make([]byte, 4)
	if _, err := rand.Read(A0); err != nil {
		return err
	}

	packet0Payload := packet.NewPacket0Payload(time.Now(), [4]byte(A0))
	err := c.send(packet.NewPacket0(packet0Payload))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Listen(ctx context.Context) {
	buf := make([]byte, 2048)
	const poll = 1 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_ = c.conn.SetReadDeadline(time.Now().Add(poll))
		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			c.report(err)
			return
		}

		pkt, err := protocol.DecodeS2CPacket(buf[:n])
		if err != nil {
			continue
		}

		if err := c.open(pkt); err != nil {
			continue
		}

		c.handlePacket(pkt)
	}
}

func (c *Client) handlePacket(pkt *protocol.S2CPacket) {
	slog.Log(context.Background(), -8 /* LevelTrace */, "Received packet", "Type", pkt.Type.String())

	var (
		msg *protocol.Message
		err error
	)
	switch pkt.Type {
	case protocol.PTInit1:
		c.handleInit1(pkt)
	case protocol.PTCommand:
		err = c.ack(pkt)
		if err != nil {
			c.report(err)
			return
		}
		msg, err = c.handleCommandPacket(pkt)
		if err != nil {
			slog.Debug("ignoring malformed command packet", "type", pkt.Type.String(), "flags", protocol.FlagsString(pkt.Flags), "pid", pkt.PId, "error", err)
			return
		}
	case protocol.PTCommandLow:
		err = c.ack(pkt)
		if err != nil {
			c.report(err)
			return
		}
		msg, err = c.handleCommandPacket(pkt)
		if err != nil {
			slog.Debug("ignoring malformed command packet", "type", pkt.Type.String(), "flags", protocol.FlagsString(pkt.Flags), "pid", pkt.PId, "error", err)
			return
		}
	case protocol.PTHeartbeat:
		err = c.ack(pkt)
		if err != nil {
			c.report(err)
			return
		}
		return
	default:
	}

	if msg != nil {
		c.recvCh <- msg
	}
}

func (c *Client) Recv(ctx context.Context) (*protocol.Message, error) {
	select {
	case p := <-c.recvCh:
		return p, nil
	case err := <-c.errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) ServerIPString() string {
	if c.raddr == nil {
		return ""
	}
	return c.raddr.IP.String()
}

func (c *Client) ack(p *protocol.S2CPacket) error {
	ackPayload := packet.NewAckPayload(p.PId)

	switch p.Type {
	case protocol.PTHeartbeat:
		return c.send(packet.NewAckHeartbeat(ackPayload))
	case protocol.PTCommandLow:
		return c.send(packet.NewAckLow(ackPayload))
	case protocol.PTCommand:
		return c.send(packet.NewAck(ackPayload))
	}

	return nil
}

func (c *Client) report(err error) {
	c.errOnce.Do(func() { c.errCh <- err })
}

func (c *Client) handleInit1(pkt *protocol.S2CPacket) {
	if len(pkt.Data) == 0 {
		c.report(fmt.Errorf("empty init payload"))
		return
	}
	step := packet.Step(pkt.Data[0])

	switch step {
	case packet.Step1:
		ok, packet1Payload := packet.DecodePacket1Payload(pkt.Data)
		if !ok {
			c.report(fmt.Errorf("bad Packet1 payload"))
			return
		}
		packet2Payload := packet.NewPacket2Payload(
			time.Now(),
			packet1Payload.A1,
			packet1Payload.A0r,
		)
		err := c.send(packet.NewPacket2(packet2Payload))
		if err != nil {
			c.report(err)
			return
		}
	case packet.Step3:
		ok, payload := packet.DecodePacket3Payload(pkt.Data)
		if !ok {
			c.report(fmt.Errorf("bad Packet3 payload"))
			return
		}
		alpha, err := handshake.GenerateAlphaB64()
		if err != nil {
			c.report(err)
			return
		}
		omega, err := handshake.OmegaB64FromPrivatePEMFile(c.config.KeyPath)
		if err != nil {
			c.report(err)
			return
		}
		keyOffset, _, err := handshake.FindKeyOffset(omega, c.config.HashcashLevel)
		if err != nil {
			c.report(err)
			return
		}
		c.Session.HandshakeState.AlphaB64 = alpha
		c.Session.HandshakeState.OmegaB64 = omega
		c.Session.HandshakeState.KeyOffset = keyOffset

		clientInitIV := message.NewClientInitIV(
			alpha,
			omega,
			1,
			c.ServerIPString(),
		)

		var Y [64]byte
		copy(Y[:], handshake.SolveRSAPuzzleY(payload.X[:], payload.N[:], payload.Level))
		packet4Payload := packet.NewPacket4Payload(
			time.Now(),
			payload.X,
			payload.N,
			Y,
			payload.A2,
			payload.Level,
			clientInitIV.Encode(),
		)
		if err := c.send(packet.NewPacket4(packet4Payload)); err != nil {
			c.report(err)
			return
		}
	}
}

func (c *Client) handleCommandPacket(p *protocol.S2CPacket) (*protocol.Message, error) {
	isFragment := p.Flags&protocol.FlagFR != 0
	isCompressed := p.Flags&protocol.FlagCP != 0

	if c.cmdFragActive && !c.cmdFragStarted.IsZero() && time.Since(c.cmdFragStarted) > commandFragmentStallTimeout {
		slog.Debug("resetting stalled command fragment stream", "expectedPID", c.cmdFragNextPID, "receivedPID", p.PId)
		c.resetCommandFragmentState()
	}

	// Non-fragmented packet while no stream is active.
	if !isFragment && !c.cmdFragActive {
		return c.parseAndHandleCommand(p.Data, isCompressed)
	}

	// Fragmented stream starts on FR packet.
	if isFragment && !c.cmdFragActive {
		c.cmdFragActive = true
		c.cmdFragCompressed = isCompressed
		c.cmdFragBuf = append(c.cmdFragBuf[:0], p.Data...)
		c.cmdFragNextPID = p.PId + 1
		c.cmdFragStarted = time.Now()
		return nil, nil
	}

	// If a stream is active, only consume strictly sequential packet IDs.
	if p.PId != c.cmdFragNextPID {
		if isFragment {
			// Start a new fragmented stream if this looks like a new first fragment.
			c.cmdFragActive = true
			c.cmdFragCompressed = isCompressed
			c.cmdFragBuf = append(c.cmdFragBuf[:0], p.Data...)
			c.cmdFragNextPID = p.PId + 1
			c.cmdFragStarted = time.Now()
			return nil, nil
		}
		// Lost/out-of-order middle fragment: abandon stale stream and drop this packet.
		// Trying to parse this as a standalone command tends to produce binary garbage.
		c.resetCommandFragmentState()
		return nil, nil
	}

	// Consume next fragment/middle packet.
	c.cmdFragBuf = append(c.cmdFragBuf, p.Data...)
	c.cmdFragNextPID = p.PId + 1

	// Middle packets are unflagged. Final packet of a fragmented stream has FR set.
	if !isFragment {
		return nil, nil
	}

	data := append([]byte(nil), c.cmdFragBuf...)
	compressed := c.cmdFragCompressed
	c.cmdFragActive = false
	c.cmdFragCompressed = false
	c.cmdFragBuf = nil
	c.cmdFragNextPID = 0
	c.cmdFragStarted = time.Time{}

	return c.parseAndHandleCommand(data, compressed)
}

func (c *Client) resetCommandFragmentState() {
	c.cmdFragActive = false
	c.cmdFragCompressed = false
	c.cmdFragBuf = nil
	c.cmdFragNextPID = 0
	c.cmdFragStarted = time.Time{}
}

func (c *Client) parseAndHandleCommand(data []byte, compressed bool) (*protocol.Message, error) {
	if compressed {
		plainData, err := protocol.QuickLZDecompressLevel1(data)
		if err != nil {
			return nil, err
		}
		data = plainData
	}

	messageEnvelope, err := protocol.ParseMessage(string(data))
	if err != nil {
		return nil, err
	}
	if !isLikelyCommandName(messageEnvelope.Name) {
		return nil, fmt.Errorf("invalid command name bytes")
	}

	slog.Debug("Received message", "Name", messageEnvelope.Name)

	if err := c.handleMessage(messageEnvelope); err != nil {
		return nil, err
	}

	return messageEnvelope, nil
}

func isLikelyCommandName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch >= 'a' && ch <= 'z' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			continue
		}
		if ch == '_' {
			continue
		}
		return false
	}
	return true
}

func (c *Client) handleMessage(messageEnvelope *protocol.Message) error {
	switch messageEnvelope.Name {
	case string(message.NameInitivexpand2):

		return c.handleInitivexpand2(messageEnvelope)
	case string(message.NameInitserver):
		return c.handleInitserver(messageEnvelope)
	}

	return nil
}

func (c *Client) handleInitivexpand2(m *protocol.Message) error {
	var paylaod message.Initivexpand2Payload
	err := protocol.DecodeMessage(m.Raw, &paylaod)
	if err != nil {
		return err
	}

	if err := handshake.VerifyInitivexpand2Proof(
		paylaod.Omega,
		paylaod.L,
		paylaod.Proof,
	); err != nil {
		return err
	}

	priv, err := handshake.PrivateKeyFromPEMFile(c.config.KeyPath)
	if err != nil {
		return err
	}

	iv, _, ek, ekProof, err := handshake.SharedIVFromInitivexpand2(
		c.Session.HandshakeState.AlphaB64,
		paylaod.Beta,
		paylaod.L,
		priv,
	)
	if err != nil {
		return err
	}
	c.SetSharedIV(iv)

	if err := c.sendClientEK(ek, ekProof); err != nil {
		return err
	}

	if err := c.sendClientInit(); err != nil {
		return err
	}
	return nil
}

func parseClientID(params map[string]string) (uint16, bool) {
	keys := []string{"aclid", "clid", "client_id"}
	for _, key := range keys {
		val, ok := params[key]
		if !ok || val == "" {
			continue
		}
		n, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			continue
		}
		return uint16(n), true
	}
	return 0, false
}

func (c *Client) sendClientEK(ek, ekProof []byte) error {
	ekMessage := message.Clientek{
		Ek:    ek,
		Proof: ekProof,
	}
	packet, err := packet.NewCommandPacket(ekMessage)
	if err != nil {
		return err
	}
	return c.send(packet)
}

func (c *Client) sendClientInit() error {
	initMessage := message.NewClientInit(
		c.config.Nickname,
		c.config.Version,
		c.config.Platform,
		c.config.VersionSign,
		c.config.HWID,
		c.Session.HandshakeState.KeyOffset,
	)
	packet, err := packet.NewCommandPacket(initMessage)
	if err != nil {
		return err
	}
	return c.send(packet)
}

func (c *Client) sendClientDisconnect(reason string) error {
	return c.SendMessage(message.NewClientDisconnect(reason))
}

func (c *Client) nextVoiceID() uint16 {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	c.voiceID++
	if c.voiceID == 0 {
		c.voiceID = 1
	}
	return c.voiceID
}

func (c *Client) handleInitserver(envelope *protocol.Message) error {
	var initserver message.InitServer
	err := protocol.DecodeMessage(envelope.Raw, &initserver)
	if err != nil {
		return err
	}

	switch {
	case initserver.ClId != 0:
		c.SetClientID(initserver.ClId)
	case initserver.ClientId != 0:
		c.SetClientID(initserver.ClientId)
	case initserver.AclId != 0:
		c.SetClientID(initserver.AclId)
	}

	return nil
}
