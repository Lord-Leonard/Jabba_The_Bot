package domain

type Track struct {
	Title          string
	Artist         string
	Album          string
	VideoID        string
	CoverUrl       string
	CoverLocalPath string
	ISRC           string
}

type WeightedSeed struct {
	Track  Track
	Weight float64
}

func NewTrackWithVideoID(title, artist, videoID, coverArtUrl string) *Track {
	return &Track{
		Title:    title,
		Artist:   artist,
		VideoID:  videoID,
		CoverUrl: coverArtUrl,
	}
}

func NewTrackWithISRC(title, Artist, coverArtUrl, ISRC string) *Track {
	return &Track{
		Title:    title,
		Artist:   Artist,
		CoverUrl: coverArtUrl,
		ISRC:     ISRC,
	}
}
