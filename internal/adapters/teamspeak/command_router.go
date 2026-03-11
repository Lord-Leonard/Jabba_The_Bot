package teamspeak

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"Jabba_The_Bot/internal/music"
	"Jabba_The_Bot/pkg/teamspeak"
)

type CommandRouter struct {
	svc *music.Service
}

func NewCommandRouter(svc *music.Service) *CommandRouter {
	return &CommandRouter{
		svc: svc,
	}
}

func (r *CommandRouter) Register(client *teamspeak.Client) {
	teamspeak.RegisterMessageHandlerTyped(client, teamspeak.MessageNameTextMessage, func(c *teamspeak.Client, textMessage teamspeak.TextMessage) {
		r.handleMessage(c, textMessage)
	})
	teamspeak.RegisterMessageHandlerTyped(client, teamspeak.MessageNameError, func(_ *teamspeak.Client, tsErr teamspeak.CommandError) {
		handleTeamSpeakError(tsErr)
	})
}

func (r *CommandRouter) handleMessage(client *teamspeak.Client, textMessage teamspeak.TextMessage) {
	msg := strings.TrimSpace(textMessage.Message)

	switch {
	case msg == "!ping":
		if err := client.Send(teamspeak.NewSendTextMessage("pong!")); err != nil {
			slog.Error("Error sending response", "error", err)
		}
	case msg == "!nowplaying" || msg == "!np":
		now := r.svc.NowPlaying()
		if now == nil {
			_ = client.Send(teamspeak.NewSendTextMessage("nothing is currently playing"))
			return
		}
		pos := formatClock(now.Position)
		dur := "?:??"
		if now.Duration > 0 {
			dur = formatClock(now.Duration)
		}
		_ = client.Send(teamspeak.NewSendTextMessage(
			fmt.Sprintf("now playing: %s (%s/%s)", now.Title, pos, dur),
		))
	case msg == "!skip":
		if r.svc.NowPlaying() == nil {
			_ = client.Send(teamspeak.NewSendTextMessage("nothing to skip"))
			return
		}
		r.svc.Skip()
		_ = client.Send(teamspeak.NewSendTextMessage("skipped current track"))
		return
	case strings.HasPrefix(msg, "!seek "):
		rawTarget := strings.TrimSpace(strings.TrimPrefix(msg, "!seek "))
		if rawTarget == "" {
			_ = client.Send(teamspeak.NewSendTextMessage("usage: !seek <seconds|mm:ss|hh:mm:ss>"))
			return
		}

		target, err := parseSeekTarget(rawTarget)
		if err != nil {
			_ = client.Send(teamspeak.NewSendTextMessage("invalid seek target; use seconds, mm:ss, or hh:mm:ss"))
			return
		}
		if err := r.svc.Seek(target); err != nil {
			_ = client.Send(teamspeak.NewSendTextMessage("seek failed: " + err.Error()))
			return
		}
		_ = client.Send(teamspeak.NewSendTextMessage(fmt.Sprintf("seeked to %s", formatClock(target))))
	case msg == "!stop":
		r.svc.Stop()
		_ = client.Send(teamspeak.NewSendTextMessage("stopped playback"))
		if err := client.UpdateNickname("Musik"); err != nil {
			slog.Error("Error resetting nickname", "error", err)
		}
		if err := client.UpdateDescription("Musik Bot"); err != nil {
			slog.Error("Error resetting description", "error", err)
		}
	case strings.HasPrefix(msg, "!play "):
		query := strings.TrimSpace(strings.TrimPrefix(msg, "!play "))
		if query == "" {
			_ = client.Send(teamspeak.NewSendTextMessage("usage: !play <query>"))
			return
		}
		go func() {
			r.svc.Stop()
			err := r.svc.Play(context.Background(), query)
			if err != nil {
				slog.Error("Manager start failed", "error", err, "query", query)
				_ = client.Send(teamspeak.NewSendTextMessage("play failed"))
				return
			}
			_ = client.Send(teamspeak.NewSendTextMessage("searching and streaming: " + query))
		}()
	case msg == "!queue" || msg == "!q":
		currentQueue := r.svc.Queue()
		if len(currentQueue) == 0 {
			_ = client.Send(teamspeak.NewSendTextMessage("queue is empty"))
			return
		}

		const maxItems = 10
		limit := len(currentQueue)
		if limit > maxItems {
			limit = maxItems
		}

		lines := make([]string, 0, limit+2)
		lines = append(lines, fmt.Sprintf("queue (%d):", len(currentQueue)))
		for i := 0; i < limit; i++ {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, describeQueueTrack(currentQueue[i])))
		}
		if len(currentQueue) > limit {
			lines = append(lines, fmt.Sprintf("... and %d more", len(currentQueue)-limit))
		}
		_ = client.Send(teamspeak.NewSendTextMessage(strings.Join(lines, "\n")))
	case strings.HasPrefix(msg, "!queue ") || strings.HasPrefix(msg, "!q "):
		query := strings.TrimSpace(strings.TrimPrefix(msg, "!queue "))
		if query == "" {
			query = strings.TrimSpace(strings.TrimPrefix(msg, "!q "))
		}
		if query == "" {
			_ = client.Send(teamspeak.NewSendTextMessage("usage: !queue <query>"))
			return
		}

		position := len(r.svc.Queue()) + 1
		track, err := r.svc.Resolve(context.Background(), query)
		if err != nil {
			_ = client.Send(teamspeak.NewSendTextMessage("queue failed: resolve error"))
			return
		}

		err = r.svc.Enqueue(track)
		if err != nil {
			_ = client.Send(teamspeak.NewSendTextMessage("queue failed: enqueue error"))
			return
		}
		_ = client.Send(teamspeak.NewSendTextMessage(fmt.Sprintf("queued at position %d: %s", position, describeQueueTrack(track))))
	case strings.HasPrefix(msg, "!search "):
		if r.svc == nil {
			_ = client.Send(teamspeak.NewSendTextMessage("search is not configured"))
			return
		}

		query := strings.TrimSpace(strings.TrimPrefix(msg, "!search "))
		if query == "" {
			_ = client.Send(teamspeak.NewSendTextMessage("usage: !search <query>"))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		results, err := r.svc.Search(ctx, query)
		if err != nil {
			slog.Error("search failed", "error", err, "query", query)
			_ = client.Send(teamspeak.NewSendTextMessage("search failed"))
			return
		}
		if len(results) == 0 {
			_ = client.Send(teamspeak.NewSendTextMessage("no results found"))
			return
		}

		lines := make([]string, 0, len(results)+1)
		lines = append(lines, fmt.Sprintf("results for: %s", query))
		for i, track := range results {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, track.Title))
		}
		_ = client.Send(teamspeak.NewSendTextMessage(strings.Join(lines, "\n")))
	}
}

func parseSeekTarget(input string) (time.Duration, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("empty seek target")
	}

	if !strings.Contains(input, ":") {
		seconds, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return 0, err
		}
		if seconds < 0 {
			seconds = 0
		}
		return time.Duration(seconds * float64(time.Second)), nil
	}

	parts := strings.Split(input, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("invalid clock format")
	}

	values := make([]int, len(parts))
	for i := range parts {
		v, err := strconv.Atoi(parts[i])
		if err != nil || v < 0 {
			return 0, fmt.Errorf("invalid clock value")
		}
		values[i] = v
	}

	var hours, minutes, seconds int
	if len(values) == 2 {
		minutes = values[0]
		seconds = values[1]
	} else {
		hours = values[0]
		minutes = values[1]
		seconds = values[2]
	}
	if seconds >= 60 || (len(values) == 3 && minutes >= 60) {
		return 0, fmt.Errorf("invalid clock range")
	}

	totalSeconds := hours*3600 + minutes*60 + seconds
	return time.Duration(totalSeconds) * time.Second, nil
}

func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func describeQueueTrack(track domain.Track) string {
	title := strings.TrimSpace(track.Title)
	artist := strings.TrimSpace(track.Artist)
	videoID := strings.TrimSpace(track.VideoID)

	if title == "" {
		title = "<unknown track>"
	}

	switch {
	case artist != "" && videoID != "":
		return fmt.Sprintf("%s - %s (%s)", title, artist, videoID)
	case artist != "":
		return fmt.Sprintf("%s - %s", title, artist)
	case videoID != "":
		return fmt.Sprintf("%s (%s)", title, videoID)
	default:
		return title
	}
}

func handleTeamSpeakError(tsErr teamspeak.CommandError) {
	if tsErr.ID == 0 {
		return
	}

	attrs := []any{"id", tsErr.ID, "msg", tsErr.Message}
	if info, ok := teamspeak.LookupErrorCode(tsErr.ID); ok {
		attrs = append(attrs, "name", info.Name)
		if info.Description != "" {
			attrs = append(attrs, "description", info.Description)
		}
	}
	if tsErr.ReturnCode != "" {
		attrs = append(attrs, "return_code", tsErr.ReturnCode)
	}
	if tsErr.FailedPermID != "" {
		attrs = append(attrs, "failed_permid", tsErr.FailedPermID)
	}
	if tsErr.ExtraMessage != "" {
		attrs = append(attrs, "extra_msg", tsErr.ExtraMessage)
	}

	slog.Error("TeamSpeak command error", attrs...)
}
