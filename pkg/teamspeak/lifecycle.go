package teamspeak

import "context"

// Start connects and starts the Teamspeak client run loop in a goroutine.
// The optional register callback can register command handlers before the loop starts.
func Start(ctx context.Context, cfg Config, register func(*Client)) (*Client, <-chan error, error) {
	c, err := CreateClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	if register != nil {
		register(c)
	}
	if err := c.Initialize(ctx); err != nil {
		return nil, nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(ctx)
		close(errCh)
	}()

	return c, errCh, nil
}

// Run connects and runs the Teamspeak client until context cancellation or error.
// The optional register callback can register command handlers before the loop starts.
func Run(ctx context.Context, cfg Config, register func(*Client)) error {
	c, err := CreateClient(cfg)
	if err != nil {
		return err
	}
	defer c.Close()

	if register != nil {
		register(c)
	}
	if err := c.Initialize(ctx); err != nil {
		return err
	}

	return c.Run(ctx)
}
