package youtube

import (
	"encoding/json"
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
		return nil, err
	}

	var playlists models.PlaylistData
	if err := json.Unmarshal(output, &playlists); err != nil {
		return nil, err
	}

	return playlists.Entries, nil
}

func FetchAllSongs(playlistURL string) ([]models.Song, error) {
	cmd := exec.Command(
		"yt-dlp",
		"--flat-playlist",
		"--dumb-single-json",
		playlistURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var songs models.SongsData
	if err := json.Unmarshal(output, &songs); err != nil {
		return nil, err
	}

	return songs.Entries, nil
}
