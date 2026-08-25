package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m model) View() tea.View {
	var view tea.View

	switch m.screen {
	case playlistScreen:
		view = m.PlaylistView()
	case songsScreen:
		view = m.SongsView()
	case queueScreen:
		view = m.QueueView()
	}

	view.AltScreen = true
	return view
}

func (m model) PlaylistView() tea.View {
	if len(m.playlists) == 0 {
		loadingPlaylists := fmt.Sprintf("%s Loading Playlists...", m.spinner.View())
		return tea.NewView(loadingPlaylists)
	}

	return tea.NewView(m.playlistsTable.View())
}

func (m model) SongsView() tea.View {
	loadingSongs := fmt.Sprintf("%s Loading Songs...", m.spinner.View())

	return tea.NewView(loadingSongs)
}

func (m model) QueueView() tea.View {
	loadingQueue := fmt.Sprintf("%s Loading Queue...", m.spinner.View())

	return tea.NewView(loadingQueue)
}
