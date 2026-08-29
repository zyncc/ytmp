package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	var content string

	switch m.screen {
	case PlaylistScreen:
		content = m.PlaylistContent()
	case SongsScreen:
		content = m.SongsContent()
	case QueueScreen:
		content = m.QueueContent()
	case KeybindsScreen:
		content = m.KeybindsContent()
	}

	playerBar := m.PlayerBarView()

	var view string
	if m.height > 0 {
		playerBarHeight := lipgloss.Height(playerBar)
		contentHeight := max(0, m.height-playerBarHeight)
		placedContent := lipgloss.PlaceVertical(contentHeight, lipgloss.Top, content)
		view = lipgloss.JoinVertical(lipgloss.Left, placedContent, playerBar)
	} else {
		view = fmt.Sprintf("%s\n\n%s", content, playerBar)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func (m Model) PlaylistContent() string {
	if m.favoritesOnly && len(m.playlists) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Subtle)).Render("No Favorite Playlists...")
	} else if len(m.playlists) == 0 {
		loadingText := lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Text)).Render("Loading Playlists...")
		return fmt.Sprintf("%s %s", m.spinner.View(), loadingText)
	}

	return m.playlistsTable.View()
}

func (m Model) SongsContent() string {
	if len(m.songs) == 0 {
		loadingText := lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Text)).Render("Loading Songs...")
		return fmt.Sprintf("%s %s", m.spinner.View(), loadingText)
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Text)).Bold(true).Render(m.selectedPlaylist.Title)
	return fmt.Sprintf("%s\n\n%s", title, m.songsTable.View())
}

func (m Model) QueueContent() string {
	if len(m.q.Songs) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Text)).Render("Queue is empty...")
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Text)).Bold(true).Render("Queue")
	return fmt.Sprintf("%s\n\n%s", title, m.queueTable.View())
}
