package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) PlayerBarView() string {
	var leftSide string
	var percent float64
	volumeIcon := " "

	switch {
	case m.volume == 0:
		volumeIcon = " "
	case m.volume < 50:
		volumeIcon = " "
	default:
		volumeIcon = " "
	}

	if !m.isPlaying || m.q.IsEmpty() {
		leftSide = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Subtle)).
			Render("Nothing playing")
		percent = 0.0
	} else {
		song := m.q.Current()
		iconSymbol := "▶"
		if m.isPaused {
			iconSymbol = "⏸"
		}
		icon := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Secondary)).
			Bold(true).
			Render(iconSymbol)
		title := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Text)).
			Bold(true).
			Render(song.Title)
		channel := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.config.Theme.Text)).
			Render(song.Channel)

		leftSide = fmt.Sprintf("%s %s • %s", icon, title, channel)
		if m.duration > 0 {
			percent = float64(m.currentTime) / float64(m.duration)
		}
	}

	volumeStr := fmt.Sprintf("%s %d", volumeIcon, m.volume)
	volumeStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.config.Theme.Text)).
		Render(volumeStr)

	targetWidth := m.width - 4
	if targetWidth <= 0 {
		targetWidth = 80
	}

	rightWidth := lipgloss.Width(volumeStyled)
	maxLeftWidth := max(0, targetWidth-rightWidth-2)
	if lipgloss.Width(leftSide) > maxLeftWidth {
		leftSide = ansi.Truncate(leftSide, maxLeftWidth, "…")
	}

	leftWidth := lipgloss.Width(leftSide)
	spaceCount := max(targetWidth-leftWidth-rightWidth, 1)
	spaces := strings.Repeat(" ", spaceCount)
	titleLine := fmt.Sprintf("%s%s%s", leftSide, spaces, volumeStyled)

	timeStr := fmt.Sprintf("%s / %s", FormatDuration(m.currentTime), FormatDuration(m.duration))
	timeStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.config.Theme.Subtle)).
		Render(timeStr)

	bar := m.progress.ViewAs(percent)
	return fmt.Sprintf("%s\n%s  %s", titleLine, timeStyled, bar)
}
