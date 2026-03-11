package metadata

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
	"database/sql"
	"errors"
)

// validate SQLiteStore implements Store
var _ Store = (*SQLiteStore)(nil)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Save(ctx context.Context, track domain.Track) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO track_metadata 
			(video_id, title, artist, album, cover_url, fetched_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(video_id) DO UPDATE SET
			LAST_ACCESSED_AT = CURRENT_TIMESTAMP
`,
		track.VideoID,
		track.Title,
		track.Artist,
		track.Album,
		track.CoverUrl,
	)
	return err
}

func (s *SQLiteStore) Get(ctx context.Context, videoID string) (domain.Track, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT video_id, title, artist, album, cover_url, cover_local_path
		FROM track_metadata
		WHERE video_id = ?
		`,
		videoID,
	)

	var t domain.Track
	var coverLocalPath sql.NullString
	if err := row.Scan(&t.VideoID, &t.Title, &t.Artist, &t.Album, &t.CoverUrl, &coverLocalPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Track{}, nil
		}
		return domain.Track{}, err
	}

	if coverLocalPath.Valid {
		t.CoverLocalPath = coverLocalPath.String
	}

	return t, nil
}

func (s *SQLiteStore) SetCoverLocalPath(ctx context.Context, videoID string, path string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE track_metadata
		SET cover_local_path = ?
		WHERE video_id = ?
	`, path, videoID)
	return err
}
