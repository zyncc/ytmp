package main

import (
	"log"
	"strconv"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/models"
	"github.com/zyncc/ytmp/youtube"
	"go.uber.org/zap"
)

type UpdatePlaylistCache struct {
	playlists []models.Playlist
	err       error
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		titleWidth := msg.Width - 14
		titleWidth = max(titleWidth, 20)
		m.playlistsTable.SetColumns([]table.Column{
			{Title: "#", Width: 4},
			{Title: "Title", Width: titleWidth},
		})
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
				return m, nil
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

		playlists, err := m.cacheRepository.FetchPlaylists()
		if err != nil {
			log.Fatal("failed to fetch playlists from cache", zap.Error(err))
		}

		m.playlists = playlists
		m.playlistsTable.SetRows(playlistsToRows(m.playlists))
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}
