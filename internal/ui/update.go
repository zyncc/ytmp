package ui

import (
	"encoding/json"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/models"
	"github.com/zyncc/ytmp/internal/mpv"
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

type SongsFetchedMsg struct {
	playlistID string
	songs      []cache.Song
	err        error
}

type StreamURLFetchedMsg struct {
	SongURL   string
	StreamURL string
	Err       error
	AutoPlay  bool
}

func fetchStreamURLCmd(song cache.Song, autoPlay bool) tea.Cmd {
	return func() tea.Msg {
		streamURL, err := youtube.FetchSong(song.URL)
		return StreamURLFetchedMsg{
			SongURL:   song.URL,
			StreamURL: streamURL,
			Err:       err,
			AutoPlay:  autoPlay,
		}
	}
}

func fetchSongsFromYTDLPCmd(cacheRepo cache.Storer, playlistID string) tea.Cmd {
	return func() tea.Msg {
		ytSongs, err := youtube.FetchAllSongs(playlistID)
		if err != nil {
			return SongsFetchedMsg{playlistID: playlistID, err: err}
		}
		if err := cacheRepo.UpsertSongs(playlistID, ytSongs); err != nil {
			return SongsFetchedMsg{playlistID: playlistID, err: err}
		}
		songs, err := cacheRepo.FetchAllSongs(playlistID)
		return SongsFetchedMsg{
			playlistID: playlistID,
			songs:      songs,
			err:        err,
		}
	}
}

func (m *Model) playCurrent() tea.Cmd {
	if m.q.IsEmpty() {
		return nil
	}

	currentSong := m.q.Current()
	m.isPlaying = true
	m.isPaused = false
	m.duration = currentSong.Duration
	m.currentTime = 0
	m.currentPlayingURL = currentSong.URL
	m.queueTable.SetCursor(m.q.Cursor)

	var cmds []tea.Cmd

	if streamURL, ok := m.urlCache[currentSong.URL]; ok && streamURL != "" {
		if err := m.mpvClient.PlaySong(streamURL); err != nil {
			m.log.Error("failed to play song with mpv", zap.Error(err))
		}
	} else {
		cmds = append(cmds, fetchStreamURLCmd(currentSong, true))
	}

	if nextSong, ok := m.q.PeekNext(); ok {
		if _, ok := m.urlCache[nextSong.URL]; !ok {
			cmds = append(cmds, fetchStreamURLCmd(nextSong, false))
		}
	}

	if prevSong, ok := m.q.PeekPrevious(); ok {
		if _, ok := m.urlCache[prevSong.URL]; !ok {
			cmds = append(cmds, fetchStreamURLCmd(prevSong, false))
		}
	}

	return tea.Batch(cmds...)
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
	case mpv.Event:
		var cmds []tea.Cmd
		cmds = append(cmds, waitForMPVEvent(m.mpvEvents))

		switch msg.Event {
		case "property-change":
			switch msg.Name {
			case "time-pos":
				var timePos float64
				if err := json.Unmarshal(msg.Data, &timePos); err == nil {
					m.currentTime = int(timePos)
				}
			case "pause":
				var paused bool
				if err := json.Unmarshal(msg.Data, &paused); err == nil {
					m.isPaused = paused
				}
			case "duration":
				var duration float64
				if err := json.Unmarshal(msg.Data, &duration); err == nil && duration > 0 {
					m.duration = int(duration)
				}
			}
		case "end-file":
			m.log.Info("song ended", zap.String("reason", msg.Reason))
			if msg.Reason == "eof" {
				if m.q.HasNext() {
					m.q.Next()
					cmds = append(cmds, m.playCurrent())
				} else {
					m.isPlaying = false
					m.currentTime = 0
				}
			}
		}

		return m, tea.Batch(cmds...)

	case StreamURLFetchedMsg:
		if msg.Err != nil {
			m.log.Error("failed to fetch stream url", zap.String("song_url", msg.SongURL), zap.Error(msg.Err))
			return m, nil
		}

		m.urlCache[msg.SongURL] = msg.StreamURL

		if msg.AutoPlay && m.currentPlayingURL == msg.SongURL {
			if err := m.mpvClient.PlaySong(msg.StreamURL); err != nil {
				m.log.Error("failed to play song with mpv", zap.Error(err))
			}
		}

		return m, nil

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
			m.volume -= m.config.Player.VolumeIncrementAmount
			if m.volume < 0 {
				m.volume = 0
			}
			m.mpvClient.SetVolume(m.volume)
			return m, nil

		case "=":
			m.volume += m.config.Player.VolumeIncrementAmount
			if m.volume > 100 {
				m.volume = 100
			}
			m.mpvClient.SetVolume(m.volume)
			return m, nil

		case "right":
			if err := m.mpvClient.Seek(5); err != nil {
				m.log.Error("failed to seek forward", zap.Error(err))
			}
			return m, nil

		case "left":
			if err := m.mpvClient.Seek(-5); err != nil {
				m.log.Error("failed to seek backwards", zap.Error(err))
			}
			return m, nil

		case "space":
			if err := m.mpvClient.TogglePause(); err != nil {
				m.log.Error("failed to toggle pause", zap.Error(err))
			}
			return m, nil

		case ".":
			if m.q.HasNext() {
				m.q.Next()
				cmd := m.playCurrent()
				return m, cmd
			}
			return m, nil

		case ",":
			if m.q.HasPrevious() {
				m.q.Previous()
				cmd := m.playCurrent()
				return m, cmd
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
				cmd := m.playCurrent()
				return m, cmd

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
				cmd := m.playCurrent()
				return m, cmd

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
				if m.q.IsEmpty() {
					return m, nil
				}
				selectedSongCursor := m.queueTable.Cursor()
				m.q.Cursor = selectedSongCursor
				cmd := m.playCurrent()
				return m, cmd

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
			m.songs = nil
			return m, fetchSongsFromYTDLPCmd(m.cacheRepository, msg.playlistID)
		}

		m.songs = songs
		m.songsTable.SetRows(songsToRows(songs))

		return m, nil

	case SongsFetchedMsg:
		if msg.err != nil {
			m.log.Error("failed to fetch songs using yt-dlp",
				zap.Error(msg.err),
			)
			return m, nil
		}

		m.log.Info("songs fetched from yt-dlp",
			zap.Int("count", len(msg.songs)),
		)

		if msg.playlistID == m.selectedPlaylist.ID {
			m.songs = msg.songs
			m.songsTable.SetRows(songsToRows(msg.songs))
		}

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
