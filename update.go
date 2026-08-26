package main

import (
	"log"
	"strconv"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/models"
	"github.com/zyncc/ytmp/youtube"
	"go.uber.org/zap"
)

type UpdatePlaylistCache struct {
	playlists []models.Playlist
	err       error
}

type UpdateSongCache struct {
	songs []models.Song
	err   error
}

type RefreshPlaylistCache struct{}

type FetchSongs struct {
	playlistID string
}

func playlistsToRows(playlists []cache.Playlist) []table.Row {
	rows := make([]table.Row, len(playlists))
	for i, playlist := range playlists {
		rows[i] = table.Row{
			strconv.Itoa(i + 1),
			playlist.Title,
		}
	}
	return rows
}

func songsToRows(songs []cache.Song) []table.Row {
	rows := make([]table.Row, len(songs))
	for i, song := range songs {
		rows[i] = table.Row{
			strconv.Itoa(i + 1),
			song.Title,
			*song.Uploader,
		}
	}
	return rows
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		titleWidth := msg.Width - 14
		titleWidth = max(titleWidth, 20)

		m.playlistsTable.SetColumns([]table.Column{
			{Title: "#", Width: 4},
			{Title: "Name", Width: titleWidth},
		})

		m.help.SetWidth(msg.Width)

		m.playlistsTable.SetWidth(msg.Width - 4)
		m.playlistsTable.SetHeight(msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		// global keybinds
		switch key {
		case "ctrl+c":
			return m, tea.Quit

		// volume control
		case "-":
			m.volume -= 10
			if m.volume < 0 {
				m.volume = 0
			}
			return m, nil
		case "=":
			m.volume += 10
			if m.volume > 100 {
				m.volume = 100
			}
			return m, nil

		case "q":
			switch m.screen {
			case playlistScreen:
				m.previousScreen = playlistScreen
			case songsScreen:
				m.previousScreen = songsScreen
			}
			m.screen = queueScreen
			return m, nil
		}

		switch m.screen {
		// playlist screen keybinds
		case playlistScreen:
			switch key {
			case "enter":
				m.screen = songsScreen
				m.previousScreen = playlistScreen
				playlistCursor := m.playlistsTable.Cursor()
				selectedPlaylist := m.playlists[playlistCursor]

				return m, func() tea.Msg {
					return FetchSongs{
						playlistID: selectedPlaylist.ID,
					}
				}
			case "ctrl+f":
				m.favoritesOnly = !m.favoritesOnly
				m.config.General.ToggleFavorites = m.favoritesOnly

				if err := config.Save(m.config); err != nil {
					m.log.Error("failed to save config", zap.Error(err))
					return m, nil
				}

				return m, func() tea.Msg {
					return RefreshPlaylistCache{}
				}
			case "f":
				selectedRowIndex := m.playlistsTable.Cursor()
				selectedPlaylist := m.playlists[selectedRowIndex]

				m.log.Info("favorite", zap.String("id", selectedPlaylist.ID))
				if err := m.cacheRepository.MarkFavoritePlaylist(selectedPlaylist.ID); err != nil {
					m.log.Error("failed to toggle favorite for playlist", zap.String("playlist_name", selectedPlaylist.Title), zap.Error(err))
				}

				return m, func() tea.Msg {
					return RefreshPlaylistCache{}
				}
			}

			var cmd tea.Cmd
			m.playlistsTable, cmd = m.playlistsTable.Update(msg)
			return m, cmd

		// songs screen keybinds
		case songsScreen:
			switch key {
			case "enter":
				return m, nil
			case "esc":
				m.screen = playlistScreen
				m.previousScreen = songsScreen
				return m, nil
			}

		// queue screen keybinds
		case queueScreen:
			switch key {
			case "esc":
				m.screen = m.previousScreen
				m.previousScreen = queueScreen
				return m, nil
			}
		}

	case FetchSongs:
		songs, err := m.cacheRepository.FetchAllSongs(msg.playlistID)
		if err != nil {
			return m, nil
		}

		if len(songs) == 0 {
			return m, func() tea.Msg {
				return songsLoadedMsg{
					songs: nil,
					err:   err,
					playlistURL,
				}
			}
		}

		m.songs = songs
		rows := songsToRows(songs)
		m.songsTable.SetRows(rows)

		return m, nil

	case songsLoadedMsg:
		if msg.err != nil || len(msg.songs) == 0 {
			m.log.Warn("songs not found in cache, fetching songs using yt-dlp")
			return m, func() tea.Msg {
				songs, err := youtube.FetchAllSongs()
				return UpdateSongCache{
					songs: songs,
					err:   err,
				}
			}
		}

	case playlistsLoadedMsg:
		if msg.err != nil || len(msg.playlists) == 0 {
			m.log.Warn("playlists not found in cache, fetching playlists using yt-dlp")
			return m, func() tea.Msg {
				playlists, err := youtube.FetchAllPlaylists()
				return UpdatePlaylistCache{
					playlists: playlists,
					err:       err,
				}
			}
		}

		m.playlists = msg.playlists
		m.playlistsTable.SetRows(playlistsToRows(m.playlists))
		return m, nil

	case UpdatePlaylistCache:
		if msg.err != nil {
			m.log.Fatal(
				"failed to fetch playlists from YouTube",
				zap.Error(msg.err),
			)
			return m, nil
		}

		if err := m.cacheRepository.UpsertPlaylists(msg.playlists); err != nil {
			log.Fatal("failed to insert playlists into cache", zap.Error(err))
		}

		playlists, err := m.cacheRepository.FetchPlaylists(m.config.General.ToggleFavorites)
		if err != nil {
			log.Fatal("failed to fetch playlists from cache", zap.Error(err))
		}

		m.playlists = playlists
		m.playlistsTable.SetRows(playlistsToRows(m.playlists))
		return m, nil

	case RefreshPlaylistCache:
		playlists, err := m.cacheRepository.FetchPlaylists(m.favoritesOnly)
		if err != nil {
			log.Fatal("failed to fetch playlists", zap.Error(err))
		}

		m.playlists = playlists

		rows := playlistsToRows(playlists)
		m.playlistsTable.SetRows(rows)
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}
