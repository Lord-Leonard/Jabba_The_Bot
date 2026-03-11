package main

import (
	"Jabba_The_Bot/internal/adapters/http"
	teamspeak2 "Jabba_The_Bot/internal/adapters/teamspeak"
	music2 "Jabba_The_Bot/internal/music"
	"Jabba_The_Bot/internal/music/autoplay"
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/metadata"
	"Jabba_The_Bot/internal/music/playback"
	"Jabba_The_Bot/internal/music/provider/youtube"
	"Jabba_The_Bot/internal/music/queue"
	"Jabba_The_Bot/internal/music/scheduler"
	"Jabba_The_Bot/internal/platform/storage/sqlite"
	"Jabba_The_Bot/pkg/teamspeak"
	youtube2 "Jabba_The_Bot/pkg/youtube"
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
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
	autoplayLog     string
	recommendLog    string
	demuxLog        string
	tsClientLog     string
}

func main() {
	loadDotenv()
	setupLogger()

	flags := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sqlite.Open(sqlite.DefaultConfig(flags.sqlitePath))
	if err != nil {
		slog.Error("Failed to open SQLite database", "error", err)
		return
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Error closing SQLite database", "error", closeErr)
		}
	}()

	metaStore := metadata.NewSQLiteStore(db)
	_ = metaStore

	var tsEngine *playback.Engine
	cfg := teamspeak.DefaultConfig()
	cfg.ServerAddr = "46.225.78.80:9987"
	cfg.KeyPath = "security/key.pem"
	cfg.Nickname = "Musik"
	tsClientLogger, err := parseScopedLogger(flags.tsClientLog)
	if err != nil {
		slog.Warn("invalid teamspeak client log level, falling back to off", "value", flags.tsClientLog, "error", err)
		tsClientLogger = nil
	}
	cfg.Logger = tsClientLogger

	client, err := teamspeak.CreateClient(cfg)
	if err != nil {
		slog.Error("Failed to start TeamSpeak adapter", "error", err)
		return
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			slog.Error("Error closing Teamspeak client", "error", closeErr)
		}
	}()

	if err := client.Initialize(ctx); err != nil {
		slog.Error("Failed to initialize TeamSpeak client", "error", err)
	}

	go func() {
		if err := client.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("teamspeak runtime error", "error", err)
		}
	}()

	recommendationLogLevel, err := youtube.ParseRecommendationLogLevel(flags.recommendLog)
	if err != nil {
		slog.Warn("invalid recommendation log level, falling back to off", "value", flags.recommendLog, "error", err)
		recommendationLogLevel = youtube.RecommendationLogOff
	}

	demuxLogger, err := parseScopedLogger(flags.demuxLog)
	if err != nil {
		slog.Warn("invalid demux log level, falling back to off", "value", flags.demuxLog, "error", err)
		demuxLogger = nil
	}
	youtubeClient := youtube2.NewClient()
	youtubeProvider, _ := youtube.NewProvider(
		"library/cache",
		"cookies.txt",
		youtubeClient,
		youtube.WithRecommendationLogLevel(recommendationLogLevel),
		youtube.WithRecommendationLogger(slog.Default()),
		youtube.WithDemuxerLogger(demuxLogger),
	)

	tsAudioAdapter := teamspeak2.NewAudioAdapter(client)

	userQueue := queue.New()
	autoplayQueue := queue.New()

	autoplayLogLevel, err := autoplay.ParseLogLevel(flags.autoplayLog)
	if err != nil {
		slog.Warn("invalid autoplay log level, falling back to off", "value", flags.autoplayLog, "error", err)
		autoplayLogLevel = autoplay.LogOff
	}

	autoplayController := autoplay.NewController(
		autoplayQueue,
		youtubeProvider,
		autoplay.WithLogLevel(autoplayLogLevel),
		autoplay.WithLogger(slog.Default()),
	)
	userQueue.Subscribe(queue.Subscription{
		OnEnqueue: func(track domain.Track) {
			autoplayController.AddTrack(track)

			updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := autoplayController.Update(updateCtx)
			if err != nil {
				slog.Error("Failed to update autoplay queue", "error", err)
			}
		},
	})

	trackScheduler := scheduler.NewScheduler(userQueue, autoplayQueue)
	//prefetcher := prefetch.NewPrefetcher(youtubeProvider)

	tsEngine = playback.NewEngine(trackScheduler, tsAudioAdapter, youtubeProvider)
	tsEngine.Start(ctx)

	svc := music2.NewMusicService(youtubeProvider, youtubeProvider, userQueue, tsEngine)

	teamspeak2.NewCommandRouter(svc).Register(client)

	sseBroker := http.NewSSEBroker()
	uiHandler := http.NewEmbeddedUIHandler()
	httpServer := http.NewServer(flags.httpAddr, svc, sseBroker, uiHandler)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	go func() {
		slog.Info("HTTP control API listening", "addr", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err != nil {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("jabba runtime stopping")
}

func parseFlags() runtimeFlags {
	f := runtimeFlags{}
	flag.BoolVar(&f.enableTeamSpeak, "enable-teamspeak", true, "enable TeamSpeak inbound adapter")
	flag.BoolVar(&f.enableDiscord, "enable-discord", false, "enable Discord inbound adapter scaffold")
	flag.BoolVar(&f.enableHTTP, "enable-http", true, "enable HTTP API adapter")
	flag.BoolVar(&f.embedUI, "embed-ui", true, "serve embedded frontend assets")
	flag.StringVar(&f.httpAddr, "http-addr", ":8080", "HTTP server listen address")
	flag.StringVar(&f.sqlitePath, "sqlite-path", "library/state/jabba.db", "SQLite file path for app state/metadata")
	flag.StringVar(&f.autoplayLog, "autoplay-log-level", "debug", "autoplay controller log level: off|info|debug")
	flag.StringVar(&f.recommendLog, "recommend-log-level", "debug", "recommendation log level: off|info|debug")
	flag.StringVar(&f.demuxLog, "demux-log-level", "off", "demuxer log level: off|error|warn|info|debug")
	flag.StringVar(&f.tsClientLog, "teamspeak-client-log-level", "off", "TeamSpeak client log level: off|error|warn|info|debug")
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

func parseScopedLogger(raw string) (*slog.Logger, error) {
	levelText := strings.ToLower(strings.TrimSpace(raw))
	switch levelText {
	case "", "off", "none", "disabled":
		return nil, nil
	}

	var level slog.Level
	switch levelText {
	case "error":
		level = slog.LevelError
	case "warn", "warning":
		level = slog.LevelWarn
	case "info":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	default:
		return nil, errors.New("unknown log level (use off|error|warn|info|debug)")
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})), nil
}
