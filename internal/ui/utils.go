package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	"github.com/zyncc/ytmp/internal/cache"
)

func FormatDuration(seconds int) string {
	min := seconds / 60
	sec := seconds % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func FormatViews(views int) string {
	if views >= 1_000_000_000 {
		return trimTrailingZero(formatCompactNumber(float64(views)/1_000_000_000, "B"))
	}
	if views >= 1_000_000 {
		return trimTrailingZero(formatCompactNumber(float64(views)/1_000_000, "M"))
	}
	if views >= 1_000 {
		return trimTrailingZero(formatCompactNumber(float64(views)/1_000, "K"))
	}
	if views <= 0 {
		return "-"
	}
	return strconv.Itoa(views)
}

func formatCompactNumber(n float64, suffix string) string {
	if n == math.Trunc(n) {
		return strconv.Itoa(int(n)) + suffix
	}
	return strconv.FormatFloat(n, 'f', 1, 64) + suffix
}

func trimTrailingZero(s string) string {
	return strings.NewReplacer(".0K", "K", ".0M", "M", ".0B", "B").Replace(s)
}

func playlistsToRows(playlists []cache.Playlist) []table.Row {
	rows := make([]table.Row, len(playlists))
	for i, playlist := range playlists {
		rows[i] = table.Row{
			playlist.Title,
		}
	}
	return rows
}

func songsToRows(songs []cache.Song) []table.Row {
	rows := make([]table.Row, len(songs))
	for i, song := range songs {
		rows[i] = table.Row{
			song.Title,
			song.Channel,
			FormatDuration(song.Duration),
			FormatViews(song.ViewCount),
		}
	}
	return rows
}
