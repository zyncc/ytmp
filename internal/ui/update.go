package ui

import (
	"context"
	"encoding/json"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/models"
	"github.com/zyncc/ytmp/internal/mpris"
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

func (m *Model) cancelInFlightFetch() {
	if m.fetchCancel != nil {
		m.fetchCancel()
		m.fetchCancel = nil
	}
}

func fetchStreamURLCmd(ctx context.Context, song cache.Song, autoPlay bool) tea.Cmd {
	return func() tea.Msg {
		streamURL, err := youtube.FetchSong(ctx, song.URL)
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

	m.cancelInFlightFetch()

	ctx, cancel := context.WithCancel(context.Background())
	m.fetchCancel = cancel

	currentSong := m.q.Current()
	m.isPlaying = true
	m.isPaused = false
	m.duration = currentSong.Duration
	m.currentTime = 0
	m.currentPlayingURL = currentSong.URL
	m.queueTable.SetCursor(m.q.Cursor)

	if m.mprisServer != nil {
		m.mprisServer.UpdateSong(currentSong, m.selectedPlaylist.Title)
		m.mprisServer.UpdatePlaybackStatus(true, false)
		m.mprisServer.UpdatePosition(0)
		m.mprisServer.UpdateDuration(currentSong.Duration)
		m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious())
	}

	var cmds []tea.Cmd

	if streamURL, ok := m.urlCache[currentSong.URL]; ok && streamURL != "" {
		if err := m.mpvClient.PlaySong(streamURL); err != nil {
			m.log.Error("failed to play song with mpv", zap.Error(err))
		}
	} else {
		cmds = append(cmds, fetchStreamURLCmd(ctx, currentSong, true))
	}

	if nextSong, ok := m.q.PeekNext(); ok {
		if _, ok := m.urlCache[nextSong.URL]; !ok {
			cmds = append(cmds, fetchStreamURLCmd(ctx, nextSong, false))
		}
	}

	if prevSong, ok := m.q.PeekPrevious(); ok {
		if _, ok := m.urlCache[prevSong.URL]; !ok {
			cmds = append(cmds, fetchStreamURLCmd(ctx, prevSong, false))
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

	keybindsHeight := max(m.height-playerBarHeight-3, 1)
	m.keybindsViewport.SetWidth(m.width - 4)
	m.keybindsViewport.SetHeight(keybindsHeight)
	m.keybindsViewport.SetContent(m.buildKeybindsContent())
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
					if m.mprisServer != nil {
						m.mprisServer.UpdatePosition(m.currentTime)
					}
				}
			case "pause":
				var paused bool
				if err := json.Unmarshal(msg.Data, &paused); err == nil {
					m.isPaused = paused
					if m.mprisServer != nil {
						m.mprisServer.UpdatePlaybackStatus(m.isPlaying, m.isPaused)
					}
				}
			case "duration":
				var duration float64
				if err := json.Unmarshal(msg.Data, &duration); err == nil && duration > 0 {
					m.duration = int(duration)
					if m.mprisServer != nil {
						m.mprisServer.UpdateDuration(m.duration)
					}
				}
			}
		case "end-file":
			m.log.Info("song ended", zap.String("reason", msg.Reason))
			if msg.Reason == "eof" {
				// repeat mode on
				if m.repeatMode {
					cmds = append(cmds, m.playCurrent())
				} else if m.q.HasNext() {
					m.q.Next()
					cmds = append(cmds, m.playCurrent())
				} else {
					m.isPlaying = false
					m.currentTime = 0
					if m.mprisServer != nil {
						m.mprisServer.UpdatePlaybackStatus(false, false)
						m.mprisServer.UpdatePosition(0)
						m.mprisServer.ClearMetadata()
						m.mprisServer.UpdateCanGo(false, m.q.HasPrevious())
					}
				}
			} else if msg.Reason == "error" {
				if m.q.HasNext() {
					m.q.Next()
					cmds = append(cmds, m.playCurrent())
				} else {
					m.isPlaying = false
					m.currentTime = 0
					if m.mprisServer != nil {
						m.mprisServer.UpdatePlaybackStatus(false, false)
						m.mprisServer.UpdatePosition(0)
						m.mprisServer.ClearMetadata()
						m.mprisServer.UpdateCanGo(false, m.q.HasPrevious())
					}
				}
			}
		}

		return m, tea.Batch(cmds...)

	case StreamURLFetchedMsg:
		if msg.Err != nil {
			m.log.Error("failed to fetch stream url", zap.String("song_url", msg.SongURL), zap.Error(msg.Err))
			if msg.AutoPlay && m.currentPlayingURL == msg.SongURL {
				if m.q.HasNext() {
					m.q.Next()
					return m, m.playCurrent()
				}
				m.isPlaying = false
				m.currentTime = 0
				if m.mprisServer != nil {
					m.mprisServer.UpdatePlaybackStatus(false, false)
					m.mprisServer.UpdatePosition(0)
					m.mprisServer.ClearMetadata()
				}
			}
			return m, nil
		}

		m.urlCache[msg.SongURL] = msg.StreamURL

		if msg.AutoPlay && m.currentPlayingURL == msg.SongURL {
			if err := m.mpvClient.PlaySong(msg.StreamURL); err != nil {
				m.log.Error("failed to play song with mpv", zap.Error(err))
			}
		}

		return m, nil

	case mpris.MsgPlayPause:
		if !m.isPlaying && !m.q.IsEmpty() {
			return m, m.playCurrent()
		}
		if err := m.mpvClient.TogglePause(); err != nil {
			m.log.Error("failed to toggle pause via MPRIS", zap.Error(err))
		}
		return m, nil

	case mpris.MsgPlay:
		if !m.isPlaying && !m.q.IsEmpty() {
			return m, m.playCurrent()
		}
		if m.isPlaying && m.isPaused {
			if err := m.mpvClient.Command("set_property", "pause", false); err != nil {
				m.log.Error("failed to resume playback via MPRIS", zap.Error(err))
			}
		}
		return m, nil

	case mpris.MsgPause:
		if m.isPlaying && !m.isPaused {
			if err := m.mpvClient.Command("set_property", "pause", true); err != nil {
				m.log.Error("failed to pause playback via MPRIS", zap.Error(err))
			}
		}
		return m, nil

	case mpris.MsgStop:
		if m.isPlaying {
			m.isPlaying = false
			m.currentTime = 0
			if err := m.mpvClient.Command("stop"); err != nil {
				m.log.Error("failed to stop playback via MPRIS", zap.Error(err))
			}
			if m.mprisServer != nil {
				m.mprisServer.UpdatePlaybackStatus(false, false)
				m.mprisServer.UpdatePosition(0)
				m.mprisServer.ClearMetadata()
				m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious())
			}
		}
		return m, nil

	case mpris.MsgNext:
		if m.q.HasNext() {
			m.q.Next()
			return m, m.playCurrent()
		}
		return m, nil

	case mpris.MsgPrevious:
		if m.currentTime <= 3 && m.q.HasPrevious() {
			m.q.Previous()
			return m, m.playCurrent()
		} else if m.isPlaying {
			if err := m.mpvClient.Command("seek", 0, "absolute"); err != nil {
				m.log.Error("failed to seek to start via MPRIS", zap.Error(err))
			}
			m.currentTime = 0
			if m.mprisServer != nil {
				m.mprisServer.UpdatePosition(0)
				m.mprisServer.EmitSeeked(0)
			}
			return m, nil
		}
		return m, nil

	case mpris.MsgSeek:
		if err := m.mpvClient.Seek(msg.OffsetSeconds); err != nil {
			m.log.Error("failed to seek via MPRIS", zap.Error(err))
		}
		newTime := m.currentTime + msg.OffsetSeconds
		if newTime < 0 {
			newTime = 0
		}
		if m.duration > 0 && newTime > m.duration {
			newTime = m.duration
		}
		m.currentTime = newTime
		if m.mprisServer != nil {
			m.mprisServer.UpdatePosition(m.currentTime)
			m.mprisServer.EmitSeeked(m.currentTime)
		}
		return m, nil

	case mpris.MsgSetPosition:
		pos := msg.PositionSeconds
		if pos < 0 {
			pos = 0
		}
		if m.duration > 0 && pos > m.duration {
			pos = m.duration
		}
		if err := m.mpvClient.Command("seek", pos, "absolute"); err != nil {
			m.log.Error("failed to set position via MPRIS", zap.Error(err))
		}
		m.currentTime = pos
		if m.mprisServer != nil {
			m.mprisServer.UpdatePosition(m.currentTime)
			m.mprisServer.EmitSeeked(m.currentTime)
		}
		return m, nil

	case mpris.MsgSetVolume:
		m.volume = msg.Volume
		if err := m.mpvClient.SetVolume(m.volume); err != nil {
			m.log.Error("failed to set volume via MPRIS", zap.Error(err))
		}
		if m.mprisServer != nil {
			m.mprisServer.UpdateVolume(m.volume)
		}
		return m, nil

	case mpris.MsgSetLoopStatus:
		m.repeatMode = (msg.LoopStatus == "Track" || msg.LoopStatus == "Playlist")
		if m.mprisServer != nil {
			m.mprisServer.UpdateRepeatMode(m.repeatMode)
		}
		return m, nil

	case mpris.MsgSetShuffle:
		if msg.Shuffle && !m.q.IsEmpty() {
			m.q.Shuffle()
			rows := songsToRows(m.q.Songs)
			m.queueTable.SetRows(rows)
			if m.mprisServer != nil {
				m.mprisServer.UpdateShuffle(true)
			}
			return m, m.playCurrent()
		} else if !msg.Shuffle {
			if m.mprisServer != nil {
				m.mprisServer.UpdateShuffle(false)
			}
		}
		return m, nil

	case mpris.MsgOpenURI:
		if msg.URI != "" {
			song := cache.Song{
				ID:    msg.URI,
				Title: msg.URI,
				URL:   msg.URI,
			}
			m.q.Enqueue(song)
			rows := songsToRows(m.q.Songs)
			m.queueTable.SetRows(rows)
			if !m.isPlaying {
				m.q.Cursor = len(m.q.Songs) - 1
				return m, m.playCurrent()
			}
			if m.mprisServer != nil {
				m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious() || m.isPlaying)
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

		case "?":
			if m.screen == KeybindsScreen {
				m.screen = m.previousScreen
				m.previousScreen = KeybindsScreen
				return m, nil
			}
			m.previousScreen = m.screen
			m.screen = KeybindsScreen
			m.updateTableDimensions()
			return m, nil

		// Volume control
		case "-":
			m.volume -= m.config.Player.VolumeIncrementAmount
			if m.volume < 0 {
				m.volume = 0
			}
			m.mpvClient.SetVolume(m.volume)
			if m.mprisServer != nil {
				m.mprisServer.UpdateVolume(m.volume)
			}
			return m, nil

		case "=", "+":
			m.volume += m.config.Player.VolumeIncrementAmount
			if m.volume > 100 {
				m.volume = 100
			}
			m.mpvClient.SetVolume(m.volume)
			if m.mprisServer != nil {
				m.mprisServer.UpdateVolume(m.volume)
			}
			return m, nil

		case "m":
			m.mpvClient.ToggleMute()
			m.isMuted = !m.isMuted
			return m, nil

		case "right":
			if err := m.mpvClient.Seek(5); err != nil {
				m.log.Error("failed to seek forward", zap.Error(err))
			}
			if m.mprisServer != nil {
				m.mprisServer.EmitSeeked(m.currentTime + 5)
			}
			return m, nil

		case "left":
			if err := m.mpvClient.Seek(-5); err != nil {
				m.log.Error("failed to seek backwards", zap.Error(err))
			}
			if m.mprisServer != nil {
				m.mprisServer.EmitSeeked(max(0, m.currentTime-5))
			}
			return m, nil

		case "space":
			if !m.isPlaying && !m.q.IsEmpty() {
				return m, m.playCurrent()
			}
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
			if m.currentTime <= 3 && m.q.HasPrevious() {
				m.q.Previous()
				cmd := m.playCurrent()
				return m, cmd
			} else if m.isPlaying {
				if err := m.mpvClient.Command("seek", 0, "absolute"); err != nil {
					m.log.Error("failed to seek backwards", zap.Error(err))
				}
				m.currentTime = 0
				if m.mprisServer != nil {
					m.mprisServer.UpdatePosition(0)
					m.mprisServer.EmitSeeked(0)
				}
				return m, nil
			}
			return m, nil

		case "r":
			m.repeatMode = !m.repeatMode
			if m.mprisServer != nil {
				m.mprisServer.UpdateRepeatMode(m.repeatMode)
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
			case KeybindsScreen:
				m.screen = m.previousScreen
				m.previousScreen = KeybindsScreen
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
				if m.mprisServer != nil {
					m.mprisServer.UpdateShuffle(false)
				}
				cmd := m.playCurrent()
				m.screen = QueueScreen
				m.previousScreen = SongsScreen
				return m, cmd

			case "e":
				if len(m.songs) == 0 {
					return m, nil
				}
				selectedSongCursor := m.songsTable.Cursor()
				if selectedSongCursor < 0 || selectedSongCursor >= len(m.songs) {
					return m, nil
				}
				selectedSong := m.songs[selectedSongCursor]
				m.q.Enqueue(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				if m.mprisServer != nil {
					m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious() || m.isPlaying)
				}
				if nextSong, ok := m.q.PeekNext(); ok {
					if _, ok := m.urlCache[nextSong.URL]; !ok {
						return m, fetchStreamURLCmd(context.Background(), nextSong, false)
					}
				}
				return m, nil

			case "a":
				if len(m.songs) == 0 {
					return m, nil
				}
				selectedSongCursor := m.songsTable.Cursor()
				if selectedSongCursor < 0 || selectedSongCursor >= len(m.songs) {
					return m, nil
				}
				selectedSong := m.songs[selectedSongCursor]
				m.q.PlayNext(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				if m.mprisServer != nil {
					m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious() || m.isPlaying)
				}
				if _, ok := m.urlCache[selectedSong.URL]; !ok {
					return m, fetchStreamURLCmd(context.Background(), selectedSong, false)
				}
				return m, nil

			case "s":
				if len(m.songs) == 0 {
					return m, nil
				}
				m.q.Clear()
				m.q.EnqueueAll(m.songs)
				m.q.Shuffle()
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				if m.mprisServer != nil {
					m.mprisServer.UpdateShuffle(true)
				}
				cmd := m.playCurrent()
				m.screen = QueueScreen
				m.previousScreen = SongsScreen
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

			case "a":
				if m.q.IsEmpty() {
					return m, nil
				}
				selectedSongCursor := m.queueTable.Cursor()
				if selectedSongCursor < 0 || selectedSongCursor >= len(m.q.Songs) {
					return m, nil
				}
				selectedSong := m.q.Songs[selectedSongCursor]
				m.q.PlayNext(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				if m.mprisServer != nil {
					m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious() || m.isPlaying)
				}
				if _, ok := m.urlCache[selectedSong.URL]; !ok {
					return m, fetchStreamURLCmd(context.Background(), selectedSong, false)
				}
				return m, nil

			case "e":
				if m.q.IsEmpty() {
					return m, nil
				}
				selectedSongCursor := m.queueTable.Cursor()
				if selectedSongCursor < 0 || selectedSongCursor >= len(m.q.Songs) {
					return m, nil
				}
				selectedSong := m.q.Songs[selectedSongCursor]
				m.q.Enqueue(selectedSong)
				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)
				if m.mprisServer != nil {
					m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious() || m.isPlaying)
				}
				if nextSong, ok := m.q.PeekNext(); ok {
					if _, ok := m.urlCache[nextSong.URL]; !ok {
						return m, fetchStreamURLCmd(context.Background(), nextSong, false)
					}
				}
				return m, nil

			case "d":
				if m.q.IsEmpty() {
					return m, nil
				}
				selectedSongCursor := m.queueTable.Cursor()
				if selectedSongCursor < 0 || selectedSongCursor >= len(m.q.Songs) {
					return m, nil
				}

				wasPlayingCurrent := (selectedSongCursor == m.q.Cursor)
				m.q.Remove(selectedSongCursor)

				rows := songsToRows(m.q.Songs)
				m.queueTable.SetRows(rows)

				if selectedSongCursor >= len(rows) {
					selectedSongCursor = len(rows) - 1
				}
				if selectedSongCursor < 0 {
					selectedSongCursor = 0
				}
				m.queueTable.SetCursor(selectedSongCursor)

				if m.q.IsEmpty() {
					m.isPlaying = false
					m.currentTime = 0
					_ = m.mpvClient.Command("stop")
					if m.mprisServer != nil {
						m.mprisServer.UpdatePlaybackStatus(false, false)
						m.mprisServer.UpdatePosition(0)
						m.mprisServer.ClearMetadata()
						m.mprisServer.UpdateCanGo(false, false)
					}
					return m, nil
				}

				if m.mprisServer != nil {
					m.mprisServer.UpdateCanGo(m.q.HasNext(), m.q.HasPrevious() || m.isPlaying)
				}

				if wasPlayingCurrent && m.isPlaying {
					return m, m.playCurrent()
				}

				if nextSong, ok := m.q.PeekNext(); ok {
					if _, ok := m.urlCache[nextSong.URL]; !ok {
						return m, fetchStreamURLCmd(context.Background(), nextSong, false)
					}
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.queueTable, cmd = m.queueTable.Update(msg)
			return m, cmd

		case KeybindsScreen:
			switch key {
			case "esc", "q", "?":
				m.screen = m.previousScreen
				m.previousScreen = KeybindsScreen
				return m, nil
			case "g", "home":
				m.keybindsViewport.GotoTop()
				return m, nil
			case "G", "end":
				m.keybindsViewport.GotoBottom()
				return m, nil
			}
			var cmd tea.Cmd
			m.keybindsViewport, cmd = m.keybindsViewport.Update(msg)
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
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

		if m.screen == KeybindsScreen {
			m.keybindsViewport, cmd = m.keybindsViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}
