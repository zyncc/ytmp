package models

type SongsData struct {
	SongsCount int    `json:"playlist_count"`
	Entries    []Song `json:"entries"`
}

type Song struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Duration  int    `json:"duration"`
	Artist    string `json:"uploader"`
	ViewCount int    `json:"view_count"`
}
