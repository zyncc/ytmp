package models

type PlaylistData struct {
	Entries []Playlist `json:"entries"`
}

type Playlist struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	URL        string      `json:"url"`
	Thumbnails []Thumbnail `json:"thumbnails"`
}

type Thumbnail struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}
