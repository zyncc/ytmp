package main

import (
	"database/sql"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/queue"
	"go.uber.org/zap"
)

type Screen int

const (
	playlistScreen Screen = iota
	songsScreen
	queueScreen
)

type playlistsLoadedMsg struct {
	playlists []cache.Playlist
	err       error
}

type songsLoadedMsg struct {
	songs []cache.Song
	err   error
}

type model struct {
	log             *zap.Logger
	config          *config.Config
	db              *sql.DB
	cacheRepository cache.Storer

	screen         Screen
	previousScreen Screen

	spinner spinner.Model

	playlistsTable table.Model
	songsTable     table.Model
	queueTable     table.Model

	playlists []cache.Playlist
	songs     []cache.Song

	q queue.Queue

	volume        int
	favoritesOnly bool

	keys KeyMap
	help help.Model
}

func NewModel(log *zap.Logger, config *config.Config, db *sql.DB, cacheRepository *cache.CacheRepository) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(config.Theme.Primary))

	playlistColumns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Title", Width: 40},
	}

	playlistsTable := table.New(
		table.WithColumns(playlistColumns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(config.Theme.Border)).BorderBottom(true).Bold(false)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color(config.Theme.Secondary)).Bold(false)
	playlistsTable.SetStyles(styles)

	return model{
		log:             log,
		config:          config,
		db:              db,
		cacheRepository: cacheRepository,

		screen:         playlistScreen,
		previousScreen: playlistScreen,

		spinner:        s,
		playlistsTable: playlistsTable,

		// config
		volume:        config.Player.Volume,
		favoritesOnly: config.General.ToggleFavorites,

		keys: keys,
		help: help.New(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			playlists, err := m.cacheRepository.FetchPlaylists(m.config.General.ToggleFavorites)
			return playlistsLoadedMsg{
				playlists: playlists,
				err:       err,
			}
		},
	)
}
