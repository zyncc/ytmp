package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type Keybind struct {
	Key         string
	Description string
}

type KeyCategory struct {
	Title    string
	Keybinds []Keybind
}

func GetKeyCategories() []KeyCategory {
	return []KeyCategory{
		{
			Title: "Playback & Audio",
			Keybinds: []Keybind{
				{Key: "Space", Description: "Toggle Play / Pause"},
				{Key: ".", Description: "Next track"},
				{Key: ",", Description: "Previous track / Restart"},
				{Key: "→ / ←", Description: "Seek forward / backward (5s)"},
				{Key: "+ / =", Description: "Increase volume"},
				{Key: "-", Description: "Decrease volume"},
				{Key: "r", Description: "Toggle repeat mode"},
			},
		},
		{
			Title: "Navigation & Scrolling",
			Keybinds: []Keybind{
				{Key: "↑ / k", Description: "Move cursor up"},
				{Key: "↓ / j", Description: "Move cursor down"},
				{Key: "g / Home", Description: "Jump to top of list"},
				{Key: "G / End", Description: "Jump to bottom of list"},
				{Key: "Ctrl+U / PgUp", Description: "Scroll page up"},
				{Key: "Ctrl+D / PgDn", Description: "Scroll page down"},
			},
		},
		{
			Title: "General & Application",
			Keybinds: []Keybind{
				{Key: "q", Description: "Toggle Queue screen"},
				{Key: "?", Description: "Toggle Keybinds help"},
				{Key: "Esc", Description: "Back / Return to previous view"},
				{Key: "Ctrl+C", Description: "Quit application"},
			},
		},
		{
			Title: "Playlists Screen",
			Keybinds: []Keybind{
				{Key: "Enter", Description: "Open playlist and load songs"},
				{Key: "f", Description: "Toggle favorite for playlist"},
				{Key: "Ctrl+F", Description: "Toggle favorites filter"},
			},
		},
		{
			Title: "Songs Screen",
			Keybinds: []Keybind{
				{Key: "Enter", Description: "Play song & queue remaining"},
				{Key: "s", Description: "Shuffle all songs & play"},
				{Key: "a", Description: "Play next (add after current)"},
				{Key: "e", Description: "Enqueue (add to end of queue)"},
				{Key: "Esc", Description: "Return to playlists screen"},
			},
		},
		{
			Title: "Queue Screen",
			Keybinds: []Keybind{
				{Key: "Enter", Description: "Jump to & play selected song"},
				{Key: "a", Description: "Duplicate to play next"},
				{Key: "e", Description: "Duplicate to queue end"},
				{Key: "Esc / q", Description: "Return to previous screen"},
			},
		},
	}
}

func (m Model) renderKeybindCategory(cat KeyCategory, width int) string {
	var primaryColor, secondaryColor, textColor string
	if m.config != nil {
		primaryColor = m.config.Theme.Primary
		secondaryColor = m.config.Theme.Secondary
		textColor = m.config.Theme.Text
	} else {
		primaryColor = "4"
		secondaryColor = "6"
		textColor = "7"
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(secondaryColor)).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(primaryColor)).
		Bold(true).
		Width(18).
		Align(lipgloss.Left)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(textColor))

	var lines []string
	lines = append(lines, headerStyle.Render(cat.Title))

	for _, kb := range cat.Keybinds {
		keyStr := keyStyle.Render(kb.Key)
		descStr := descStyle.Render(kb.Description)
		lines = append(lines, fmt.Sprintf("  %s  %s", keyStr, descStr))
	}

	block := strings.Join(lines, "\n")
	if width > 0 {
		return lipgloss.NewStyle().Width(width).Render(block)
	}
	return block
}

func (m Model) buildKeybindsContent() string {
	targetWidth := m.width - 4
	if targetWidth <= 0 {
		targetWidth = 80
	}

	categories := GetKeyCategories()
	var blocks []string
	for _, cat := range categories {
		blocks = append(blocks, m.renderKeybindCategory(cat, targetWidth))
	}
	return strings.Join(blocks, "\n\n")
}

func (m Model) KeybindsContent() string {
	var textColor, subtleColor string
	if m.config != nil {
		textColor = m.config.Theme.Text
		subtleColor = m.config.Theme.Subtle
	} else {
		textColor = "7"
		subtleColor = "8"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(textColor)).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(subtleColor))

	var scrollInfo string
	if m.keybindsViewport.TotalLineCount() > m.keybindsViewport.Height() && m.keybindsViewport.Height() > 0 {
		scrollPercent := int(m.keybindsViewport.ScrollPercent() * 100)
		scrollInfo = fmt.Sprintf("  (↑/↓ to scroll • %d%%)", scrollPercent)
	}

	header := fmt.Sprintf("%s  %s%s",
		titleStyle.Render("Keybinds & Controls"),
		subtitleStyle.Render("(Press '?' or 'Esc' to return)"),
		subtitleStyle.Render(scrollInfo),
	)

	return fmt.Sprintf("%s\n\n%s", header, m.keybindsViewport.View())
}
