package youtube

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/zyncc/ytmp/internal/models"
)

func FetchAllPlaylists() ([]models.Playlist, error) {
	cmd := exec.Command(
		"yt-dlp",
		"--flat-playlist",
		"--extractor-args", "youtubetab:skip=authcheck",
		"--dump-single-json",
		"https://www.youtube.com/feed/playlists",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlists: %w", err)
	}

	var playlists models.PlaylistData
	if err := json.Unmarshal(output, &playlists); err != nil {
		return nil, fmt.Errorf("failed to decode playlists json: %w", err)
	}

	return playlists.Entries, nil
}

func FetchAllSongs(playlistID string) ([]models.Song, error) {
	url := fmt.Sprintf("https://music.youtube.com/playlist?list=%s", playlistID)
	cmd := exec.Command(
		"yt-dlp",
		"--flat-playlist",
		"--dump-single-json",
		url,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp failed for %s: %w\n%s", url, err, output)
	}

	var songs models.SongsData
	if err := json.Unmarshal(output, &songs); err != nil {
		return nil, fmt.Errorf("failed to decode songs json: %w", err)
	}

	return songs.Entries, nil
}

func FetchSong(url string) (string, error) {
	cmd := exec.Command(
		"yt-dlp",
		"-f", "ba",
		"-g",
		url,
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
