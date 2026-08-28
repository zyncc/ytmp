package ui

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/models"
	"github.com/zyncc/ytmp/internal/youtube"
	"go.uber.org/zap"
)

type UpdatePlaylistCache struct {
	playlists []models.Playlist
	err       error
}

type RefreshPlaylistCache struct{}

type FetchSongs struct {
	playlistID string
}

func (m *Model) updateTableDimensions() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	const (
		durationWidth = 10
		viewsWidth    = 10
		overhead      = 12 // border and padding overhead
	)

	// Responsive artist column: proportional to terminal width, bounded between 18 and 35
	artistWidth := max(min(m.width/4, 35), 18)

	playlistTitleWidth := max(m.width-overhead, 20)
	songTitleWidth := max(m.width-artistWidth-durationWidth-viewsWidth-overhead, 20)

	m.playlistsTable.SetColumns([]table.Column{
		{Title: "Playlists", Width: playlistTitleWidth},
	})

	m.songsTable.SetColumns([]table.Column{
		{Title: "Title", Width: songTitleWidth},
		{Title: "Artist", Width: artistWidth},
		{Title: "Duration", Width: durationWidth},
		{Title: "Views", Width: viewsWidth},
	})

	m.queueTable.SetColumns([]table.Column{
		{Title: "Title", Width: songTitleWidth},
		{Title: "Artist", Width: artistWidth},
		{Title: "Duration", Width: durationWidth},
		{Title: "Views", Width: viewsWidth},
	})

	m.playlistsTable.SetWidth(m.width - 4)
	m.songsTable.SetWidth(m.width - 4)
	m.queueTable.SetWidth(m.width - 4)

	const timeStrWidth = 15 // "00:00 / 00:00  "
	barWidth := max(m.width-timeStrWidth-4, 20)
	m.progress.SetWidth(barWidth)

	playerBarHeight := lipgloss.Height(m.PlayerBarView())

	playlistTableHeight := max(m.height-playerBarHeight-1, 1)
	m.playlistsTable.SetHeight(playlistTableHeight)

	songsTableHeight := max(m.height-playerBarHeight-3, 1)
	m.songsTable.SetHeight(songsTableHeight)

	queueTableHeight := max(m.height-playerBarHeight-3, 1)
	m.queueTable.SetHeight(queueTableHeight)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableDimensions()
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		// Global keybinds
		switch key {
		case "ctrl+c":
			return m, tea.Quit

		// Volume control
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
			case PlaylistScreen:
				m.previousScreen = PlaylistScreen
				m.screen = QueueScreen
				m.updateTableDimensions()
				return m, nil
			case SongsScreen:
				m.previousScreen = SongsScreen
				m.screen = QueueScreen
				m.updateTableDimensions()
				return m, nil
			case QueueScreen:
				m.screen = m.previousScreen
				m.previousScreen = QueueScreen
				m.updateTableDimensions()
				return m, nil
			}
		}

		switch m.screen {
		// Playlist screen keybinds
		case PlaylistScreen:
			switch key {
			case "enter":
				if len(m.playlists) == 0 {
					return m, nil
				}
				m.screen = SongsScreen
				m.previousScreen = PlaylistScreen
				m.updateTableDimensions()
				playlistCursor := m.playlistsTable.Cursor()
				selectedPlaylist := m.playlists[playlistCursor]
				m.selectedPlaylist = selectedPlaylist

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
				if len(m.playlists) == 0 {
					return m, nil
				}
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

		// Songs screen keybinds
		case SongsScreen:
			switch key {
			case "enter":
				if len(m.songs) == 0 {
					return m, nil
				}
				selectedSongCursor := m.songsTable.Cursor()
				m.q.Clear()
				m.q.EnqueueAll(m.songs[selectedSongCursor:])
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				if !m.q.IsEmpty() {
					current := m.q.Current()
					m.isPlaying = true
					m.duration = current.Duration
					m.currentTime = 0
				}
				return m, nil

			case "e":
				selectedSongCursor := m.songsTable.Cursor()
				selectedSong := m.songs[selectedSongCursor]
				m.q.Enqueue(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)

			case "a":
				selectedSongCursor := m.songsTable.Cursor()
				selectedSong := m.songs[selectedSongCursor]
				m.q.PlayNext(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)

			case "s":
				if len(m.songs) == 0 {
					return m, nil
				}
				m.q.Clear()
				m.q.EnqueueAll(m.songs)
				m.q.Shuffle()
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				// if !m.q.IsEmpty() {
				// 	current := m.q.Current()
				// 	m.isPlaying = true
				// 	m.duration = current.Duration
				// 	m.currentTime = 0
				// }
				return m, nil
			case "esc":
				m.screen = PlaylistScreen
				m.previousScreen = SongsScreen
				m.songs = nil
				m.updateTableDimensions()
				return m, nil
			}
			var cmd tea.Cmd
			m.songsTable, cmd = m.songsTable.Update(msg)
			return m, cmd

		// Queue screen keybinds
		case QueueScreen:
			switch key {
			case "esc":
				m.screen = m.previousScreen
				m.previousScreen = QueueScreen
				m.updateTableDimensions()
				return m, nil

			case "q":
				m.screen = m.previousScreen
				m.previousScreen = QueueScreen
				m.updateTableDimensions()
				return m, nil

			case "enter":
				selectedSongCursor := m.queueTable.Cursor()
				m.q.Cursor = selectedSongCursor

			case "e":
				selectedSongCursor := m.queueTable.Cursor()
				selectedSong := m.q.Songs[selectedSongCursor]
				m.q.Enqueue(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)

			case "a":
				selectedSongCursor := m.queueTable.Cursor()
				selectedSong := m.q.Songs[selectedSongCursor]
				m.q.PlayNext(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)

				if m.queueTable.Cursor() != 0 {
					m.queueTable.SetCursor(m.queueTable.Cursor() + 1)
				}
			}
			var cmd tea.Cmd
			m.queueTable, cmd = m.queueTable.Update(msg)
			return m, cmd
		}

	case FetchSongs:
		m.log.Info("FetchSongs received",
			zap.String("playlist_id", msg.playlistID),
		)

		songs, err := m.cacheRepository.FetchAllSongs(msg.playlistID)
		if err != nil {
			m.log.Error("error fetching songs from cache",
				zap.Error(err),
			)
			return m, nil
		}

		m.log.Info("songs fetched from cache",
			zap.Int("count", len(songs)),
		)

		if len(songs) == 0 {
			m.log.Warn("no songs found in cache, fetching from yt-dlp")

			ytSongs, err := youtube.FetchAllSongs(msg.playlistID)
			if err != nil {
				m.log.Fatal("failed to fetch songs using yt-dlp",
					zap.Error(err),
				)
			}

			m.log.Info("songs fetched from yt-dlp",
				zap.Int("count", len(ytSongs)),
			)

			if err := m.cacheRepository.UpsertSongs(msg.playlistID, ytSongs); err != nil {
				m.log.Fatal("failed to update cache with songs",
					zap.Error(err),
				)
			}

			return m, func() tea.Msg {
				return FetchSongs{
					playlistID: msg.playlistID,
				}
			}
		}

		m.songs = songs
		m.songsTable.SetRows(songsToRows(songs))

		return m, nil

	case playlistsLoadedMsg:
		if msg.err != nil || len(msg.playlists) == 0 {
			m.log.Warn("playlists not found in cache, fetching playlists using yt-dlp", zap.Error(msg.err))
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
			m.log.Fatal("failed to insert playlists into cache", zap.Error(err))
		}

		playlists, err := m.cacheRepository.FetchPlaylists(m.config.General.ToggleFavorites)
		if err != nil {
			m.log.Fatal("failed to fetch playlists from cache", zap.Error(err))
		}

		m.playlists = playlists
		m.playlistsTable.SetRows(playlistsToRows(m.playlists))
		return m, nil

	case RefreshPlaylistCache:
		playlists, err := m.cacheRepository.FetchPlaylists(m.favoritesOnly)
		if err != nil {
			m.log.Fatal("failed to fetch playlists", zap.Error(err))
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
