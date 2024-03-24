package state

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type MetadataRecord struct {
	VideoID        string
	Provider       string
	Title          string
	Artist         string
	Album          string
	CoverURL       string
	DurationSec    int64
	FetchedAt      time.Time
	LastAccessedAt time.Time
	SourceVersion  int
}

type SQLiteMetadataStore struct {
	db *sql.DB
}

func NewSQLiteMetadataStore(db *sql.DB) *SQLiteMetadataStore {
	return &SQLiteMetadataStore{db: db}
}

func (s *SQLiteMetadataStore) GetByVideoID(ctx context.Context, videoID string) (MetadataRecord, bool, error) {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return MetadataRecord{}, false, errors.New("videoID is required")
	}

	var rec MetadataRecord
	err := s.db.QueryRowContext(
		ctx,
		`SELECT video_id, provider, title, artist, album, cover_url, duration_sec, fetched_at, last_accessed_at, source_version
		 FROM track_metadata
		 WHERE video_id = ?`,
		videoID,
	).Scan(
		&rec.VideoID,
		&rec.Provider,
		&rec.Title,
		&rec.Artist,
		&rec.Album,
		&rec.CoverURL,
		&rec.DurationSec,
		&rec.FetchedAt,
		&rec.LastAccessedAt,
		&rec.SourceVersion,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return MetadataRecord{}, false, nil
	}
	if err != nil {
		return MetadataRecord{}, false, err
	}

	return rec, true, nil
}

func (s *SQLiteMetadataStore) Upsert(ctx context.Context, rec MetadataRecord) error {
	rec.VideoID = strings.TrimSpace(rec.VideoID)
	rec.Provider = strings.TrimSpace(rec.Provider)
	if rec.VideoID == "" {
		return errors.New("videoID is required")
	}
	if rec.Provider == "" {
		rec.Provider = "youtube"
	}
	if rec.FetchedAt.IsZero() {
		rec.FetchedAt = time.Now().UTC()
	}
	if rec.LastAccessedAt.IsZero() {
		rec.LastAccessedAt = rec.FetchedAt
	}
	if rec.SourceVersion <= 0 {
		rec.SourceVersion = 1
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO track_metadata
		 (video_id, provider, title, artist, album, cover_url, duration_sec, fetched_at, last_accessed_at, source_version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(video_id) DO UPDATE SET
		   provider = excluded.provider,
		   title = excluded.title,
		   artist = excluded.artist,
		   album = excluded.album,
		   cover_url = excluded.cover_url,
		   duration_sec = excluded.duration_sec,
		   fetched_at = excluded.fetched_at,
		   last_accessed_at = excluded.last_accessed_at,
		   source_version = excluded.source_version`,
		rec.VideoID,
		rec.Provider,
		rec.Title,
		rec.Artist,
		rec.Album,
		rec.CoverURL,
		rec.DurationSec,
		rec.FetchedAt,
		rec.LastAccessedAt,
		rec.SourceVersion,
	)
	return err
}

func (s *SQLiteMetadataStore) Touch(ctx context.Context, videoID string, accessedAt time.Time) error {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return errors.New("videoID is required")
	}
	if accessedAt.IsZero() {
		accessedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		"UPDATE track_metadata SET last_accessed_at = ? WHERE video_id = ?",
		accessedAt,
		videoID,
	)
	return err
}
