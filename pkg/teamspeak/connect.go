package teamspeak

import (
	"context"
	"fmt"
	"log/slog"

	"Jabba_The_Bot/pkg/teamspeak/internal/client"
)

type Config struct {
	ServerAddr    string
	KeyPath       string
	Nickname      string
	Version       string
	VersionSign   string
	Platform      string
	HWID          string
	HashcashLevel int
	Logger        *slog.Logger
}

func DefaultConfig() Config {
	return Config{
		Version:       "3.?.? [Build: 5680278000]",
		VersionSign:   "DX5NIYLvfJEUjuIbCidnoeozxIDRRkpq3I9vVMBmE9L2qnekOoBzSenkzsg2lC9CMv8K5hkEzhr2TYUYSwUXCg==",
		Platform:      "Windows",
		HWID:          "+LyYqbDqOvEEpN5pdAbF8/v5kZ0=",
		HashcashLevel: 8,
		Logger:        slog.Default(),
	}
}

func mapConfig(cfg Config) client.Config {
	return client.Config{
		KeyPath:       cfg.KeyPath,
		Nickname:      cfg.Nickname,
		Version:       cfg.Version,
		VersionSign:   cfg.VersionSign,
		Platform:      cfg.Platform,
		HWID:          cfg.HWID,
		HashcashLevel: cfg.HashcashLevel,
		Logger:        cfg.Logger,
	}
}

// CreateClient dials the server and returns a client without starting handshake.
func CreateClient(cfg Config) (*Client, error) {
	if cfg.ServerAddr == "" {
		return nil, fmt.Errorf("server address is required")
	}

	transport, err := client.Connect(cfg.ServerAddr, mapConfig(cfg))
	if err != nil {
		return nil, err
	}

	return &Client{c: transport}, nil
}

// Connect is a compatibility helper: CreateClient + Initialize.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	c, err := CreateClient(cfg)
	if err != nil {
		return nil, err
	}

	if err := c.Initialize(ctx); err != nil {
		return nil, err
	}

	return c, nil
}
