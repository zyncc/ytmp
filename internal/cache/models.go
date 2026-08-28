package cache

import "time"

type Playlist struct {
	ID           string
	Title        string
	URL          string
	ThumbnailURL *string
	Favorite     bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Song struct {
	ID           string
	PlaylistID   string
	Title        string
	URL          string
	Duration     int
	Channel      string
	ThumbnailURL string
	ViewCount    int

	CreatedAt time.Time
	UpdatedAt time.Time
}
