package main

import (
	discordin "Jabba_The_Bot/internal/adapters/inbound/discord"
	httpadapter "Jabba_The_Bot/internal/adapters/inbound/http"
	teamspeakin "Jabba_The_Bot/internal/adapters/inbound/teamspeak"
	eventbusadapter "Jabba_The_Bot/internal/adapters/outbound/eventbus"
	outboundmusic "Jabba_The_Bot/internal/adapters/outbound/music"
	stateadapter "Jabba_The_Bot/internal/adapters/outbound/state"
	"Jabba_The_Bot/internal/core/app/ports"
	"Jabba_The_Bot/internal/core/app/service"
	"Jabba_The_Bot/internal/music"
	"Jabba_The_Bot/internal/music/output"
	"Jabba_The_Bot/internal/music/provider"
	sqliteplatform "Jabba_The_Bot/internal/platform/storage/sqlite"
	"Jabba_The_Bot/pkg/teamspeak"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

type runtimeFlags struct {
	enableTeamSpeak bool
	enableDiscord   bool
	enableHTTP      bool
	embedUI         bool
	httpAddr        string
	sqlitePath      string
}

func main() {
	loadDotenv()
	setupLogger()

	flags := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sqliteCfg := sqliteplatform.DefaultConfig(flags.sqlitePath)
	db, err := sqliteplatform.Open(sqliteCfg)
	if err != nil {
		slog.Error("Failed to initialize sqlite", "path", flags.sqlitePath, "error", err)
		return
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Failed to close sqlite", "error", closeErr)
		}
	}()
	slog.Info("SQLite initialized", "path", flags.sqlitePath)

	ytProvider := provider.NewYouTubeProvider()
	searchService := service.NewSearchService(outboundmusic.NewYouTubeSearchProvider(nil))
	appEvents := eventbusadapter.NewInProcBus()
	metadataStore := stateadapter.NewSQLiteMetadataStore(db)
	_ = metadataStore
	var manager *music.Manager

	if flags.enableTeamSpeak {
		client, m, err := startTeamSpeak(ctx, ytProvider, searchService, appEvents)
		if err != nil {
			slog.Error("Failed to start TeamSpeak adapter", "error", err)
			return
		}
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				slog.Error("Error closing Teamspeak client", "error", closeErr)
			}
		}()
		manager = m
	}

	if flags.enableDiscord {
		discordin.NewCommandRouter().Register()
	}

	if flags.enableHTTP {
		if manager == nil {
			slog.Error("HTTP adapter requires an initialized music manager")
			return
		}

		bridge := httpadapter.NewLegacyManagerBridge(manager)
		sseBroker := httpadapter.NewSSEBroker()
		httpadapter.BridgeAppEventsToSSE(ctx, appEvents, sseBroker)
		uiHandler := httpadapter.NewEmbeddedUIHandler()
		if !flags.embedUI {
			uiHandler = nil
		}
		httpServer := &http.Server{
			Addr:    flags.httpAddr,
			Handler: httpadapter.NewServerWithUI(bridge, bridge, searchService, sseBroker, uiHandler).Routes(),
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(shutdownCtx)
		}()

		go func() {
			slog.Info("HTTP control API listening", "addr", httpServer.Addr)
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("HTTP server failed", "error", err)
			}
		}()
	}

	<-ctx.Done()
	slog.Info("jabba runtime stopping")
}

func startTeamSpeak(
	ctx context.Context,
	ytProvider *provider.YouTubeProvider,
	search ports.MusicSearchQueries,
	appEvents eventbusadapter.Publisher,
) (*teamspeak.Client, *music.Manager, error) {
	cfg := teamspeak.DefaultConfig()
	cfg.ServerAddr = "46.225.78.80:9987"
	cfg.KeyPath = "security/key.pem"
	cfg.Nickname = "Musik"

	slog.Info("Starting Teamspeak client", "server", cfg.ServerAddr)
	client, err := teamspeak.CreateClient(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create teamspeak client: %w", err)
	}

	manager := music.NewManager(
		ytProvider,
		output.NewTeamSpeak(client),
		music.WithEventSink(music.EventSinkFunc(func(event music.Event) {
			if appEvents == nil {
				return
			}
			appEvents.Publish(event)
		})),
	)
	manager.SetOnFirstBytes(func(_ string, title string, duration time.Duration, cacheHit bool) {
		origin := "network"
		if cacheHit {
			origin = "cache"
		}
		_ = client.Send(teamspeak.NewSendTextMessage(fmt.Sprintf("playback started (%s, %.1fs) - %s", origin, duration.Seconds(), title)))
		if err := client.UpdateNickname("Musik ▶ " + title); err != nil {
			slog.Error("Error updating nickname", "error", err)
		}
		if err := client.UpdateDescription("Now Playing: " + title); err != nil {
			slog.Error("Error updating description", "error", err)
		}
	})

	teamspeakin.NewCommandRouter(manager, search).Register(client)

	if err := client.Initialize(ctx); err != nil {
		return nil, nil, fmt.Errorf("initialize teamspeak client: %w", err)
	}

	go func() {
		if err := client.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("teamspeak runtime error", "error", err)
		}
	}()

	return client, manager, nil
}

func parseFlags() runtimeFlags {
	f := runtimeFlags{}
	flag.BoolVar(&f.enableTeamSpeak, "enable-teamspeak", true, "enable TeamSpeak inbound adapter")
	flag.BoolVar(&f.enableDiscord, "enable-discord", false, "enable Discord inbound adapter scaffold")
	flag.BoolVar(&f.enableHTTP, "enable-http", true, "enable HTTP API adapter")
	flag.BoolVar(&f.embedUI, "embed-ui", true, "serve embedded frontend assets")
	flag.StringVar(&f.httpAddr, "http-addr", ":8080", "HTTP server listen address")
	flag.StringVar(&f.sqlitePath, "sqlite-path", "library/state/jabba.db", "SQLite file path for app state/metadata")
	flag.Parse()
	return f
}

func setupLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
}

func loadDotenv() {
	_ = godotenv.Load()
}
