package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func (m Model) PlayerBarView() string {
	var titleLine string
	var percent float64
	volumeIcon := "󰕾"

	switch {
	case m.volume == 0:
		volumeIcon = " "
	case m.volume < 50:
		volumeIcon = " "
	default:
		volumeIcon = " "
	}

	if !m.isPlaying || m.q.IsEmpty() {
		titleLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Subtle)).
			Render("Nothing playing")
		percent = 0.0
	} else {
		song := m.q.Current()
		icon := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Secondary)).
			Bold(true).
			Render("▶")
		title := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Text)).
			Bold(true).
			Render(song.Title)
		channel := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Text)).
			Render(song.Channel)

		titleLine = fmt.Sprintf("%s %s • %s      %s %d", icon, title, channel, volumeIcon, m.volume)
		if m.duration > 0 {
			percent = float64(m.currentTime) / float64(m.duration)
		}
	}

	timeStr := fmt.Sprintf("%s / %s", FormatDuration(m.currentTime), FormatDuration(m.duration))
	timeStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.config.Theme.Subtle)).
		Render(timeStr)

	bar := m.progress.ViewAs(percent)
	return fmt.Sprintf("%s\n%s  %s", titleLine, timeStyled, bar)
}
