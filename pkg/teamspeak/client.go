package teamspeak

import (
	"Jabba_The_Bot/pkg/teamspeak/internal/protocol"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"Jabba_The_Bot/pkg/teamspeak/internal/client"
)

// Client is the public-facing Teamspeak client handle.
// Low-level transport and protocol details are hidden in internal packages.
type Client struct {
	c        *client.Client
	mu       sync.RWMutex
	handlers map[MessageName]func(protocol.Message)
}

type OpusFrameSource interface {
	ProvideFrame() ([]byte, error)
}

const (
	CodecOpusVoice = protocol.CodecOpusVoice
	CodecOpusMusic = protocol.CodecOpusMusic
)

const opusFrameInterval = 20 * time.Millisecond

func (c *Client) Initialize(ctx context.Context) error {
	if c == nil || c.c == nil {
		return nil
	}
	return c.c.Initialize(ctx)
}

func (c *Client) Close() error {
	if c == nil || c.c == nil {
		return nil
	}
	return c.c.Close()
}

// RegisterMessageHandler registers a handler for a server command by name.
func (c *Client) registerMessageHandler(name MessageName, fn func(protocol.Message)) {
	if c == nil || name == "" || fn == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handlers == nil {
		c.handlers = make(map[MessageName]func(protocol.Message))
	}
	c.handlers[name] = fn
}

// Run starts the main receive loop and dispatches decrypted commands.
func (c *Client) Run(ctx context.Context) error {
	if c == nil || c.c == nil {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			err := c.Close()
			if err != nil {
				return err
			}
			return ctx.Err()
		default:
		}

		msg, err := c.c.Recv(ctx)
		if err != nil {
			return err
		}
		if msg == nil || msg.Name == "" {
			continue
		}

		c.dispatchCommand(*msg)
	}
}

func (c *Client) dispatchCommand(cmd protocol.Message) {
	c.mu.RLock()
	fn := c.handlers[MessageName(cmd.Name)]
	c.mu.RUnlock()
	if fn != nil {
		fn(cmd)
	}
}

// RegisterMessageHandlerTyped registers a handler for `name` that receives a strongly typed message T.
// T is decoded from cmd.Params using `teamspeak:"..."` tags.
func RegisterMessageHandlerTyped[T any](c *Client, name MessageName, fn func(*Client, T)) {
	if c == nil || name == "" || fn == nil {
		return
	}

	c.registerMessageHandler(name, func(envelope protocol.Message) {
		var msg T
		if err := protocol.DecodeMessage(envelope.Raw, &msg); err != nil {
			// TODO: policy choice: ignore/log/route to error handler
			return
		}
		fn(c, msg)
	})
}

func (c *Client) Send(message any) error {
	return c.c.SendMessage(message)
}

func (c *Client) SendOpusFrame(frame []byte) error {
	return c.SendOpusFrameWithCodec(frame, CodecOpusMusic)
}

func (c *Client) SendOpusFrameWithCodec(frame []byte, codec uint8) error {
	if c == nil || c.c == nil {
		return errors.New("teamspeak client is nil")
	}
	if frame == nil {
		frame = []byte{}
	}
	return c.c.SendVoiceFrame(codec, frame)
}

func (c *Client) SendOpusStream(source OpusFrameSource) error {
	return c.SendOpusStreamWithCodec(source, CodecOpusMusic)
}

func (c *Client) SendOpusStreamWithCodec(source OpusFrameSource, codec uint8) error {
	if c == nil || c.c == nil {
		return errors.New("teamspeak client is nil")
	}
	if source == nil {
		return errors.New("opus source is nil")
	}

	nextTick := time.Now()
	for {
		frame, err := source.ProvideFrame()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Empty frame marks end of stream for TeamSpeak voice.
				return c.c.SendVoiceFrame(codec, nil)
			}
			return err
		}

		if err := c.c.SendVoiceFrame(codec, frame); err != nil {
			if isVoiceFrameTooLargeError(err) {
				slog.Warn("dropping oversized opus frame", "error", err, "size", len(frame))
				nextTick = nextTick.Add(opusFrameInterval)
				if sleepFor := time.Until(nextTick); sleepFor > 0 {
					time.Sleep(sleepFor)
				} else {
					nextTick = time.Now()
				}
				continue
			}
			return err
		}

		nextTick = nextTick.Add(opusFrameInterval)
		if sleepFor := time.Until(nextTick); sleepFor > 0 {
			time.Sleep(sleepFor)
		} else {
			// If sending falls behind, reset to now to avoid runaway drift.
			nextTick = time.Now()
		}
	}
}

func isVoiceFrameTooLargeError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "opus frame too large for TeamSpeak packet")
}

func (c *Client) updateNickname(newNickname string) error {
	updateNicknameMessage := ClientUpdate{
		Nickname: &newNickname,
	}

	return c.c.SendMessage(updateNicknameMessage)
}

func (c *Client) UpdateNickname(newNickname string) error {
	if len(newNickname) < 30 {
		return c.updateNickname(newNickname)
	}

	return c.updateNickname(newNickname[:29])
}

func (c *Client) UpdateDescription(s string) error {
	if c == nil || c.c == nil {
		return errors.New("teamspeak client is nil")
	}
	clid := c.c.ClientID()
	if clid == 0 {
		return fmt.Errorf("client id not initialized yet")
	}

	editMessage := ClientEdit{
		ClientId:          clid,
		ClientDescription: &s,
	}

	return c.c.SendMessage(editMessage)
}
