package main

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Help  key.Binding
	Quit  key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Help, k.Quit},
	}
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

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
	if m.favoritesOnly && len(m.playlists) == 0 {
		noFavoritePlaylists := "No Favorite Playlists..."
		return tea.NewView(noFavoritePlaylists)
	}
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
