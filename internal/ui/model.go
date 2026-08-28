package ui

import (
	"database/sql"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/queue"
	"go.uber.org/zap"
)

// Screen represents the active UI screen.
type Screen int

const (
	PlaylistScreen Screen = iota
	SongsScreen
	QueueScreen
)

type playlistsLoadedMsg struct {
	playlists []cache.Playlist
	err       error
}

// Model represents the main Bubble Tea application state.
type Model struct {
	log             *zap.Logger
	config          *config.Config
	db              *sql.DB
	cacheRepository cache.Storer

	screen         Screen
	previousScreen Screen

	spinner  spinner.Model
	progress progress.Model

	currentTime int
	duration    int
	isPlaying   bool

	playlistsTable table.Model
	songsTable     table.Model
	queueTable     table.Model

	playlists []cache.Playlist
	songs     []cache.Song

	selectedPlaylist cache.Playlist

	q queue.Queue

	volume        int
	favoritesOnly bool

	width  int
	height int
}

// New creates and initializes a new UI Model.
func New(log *zap.Logger, config *config.Config, db *sql.DB, cacheRepository cache.Storer) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(config.Theme.Primary))

	prog := progress.New(
		progress.WithoutPercentage(),
		progress.WithFillCharacters('━', '─'),
		progress.WithColors(lipgloss.Color(config.Theme.Secondary)),
	)
	prog.EmptyColor = lipgloss.Color(config.Theme.Subtle)

	playlistsTable := table.New(
		table.WithFocused(true),
		table.WithHeight(15),
	)

	songsTable := table.New(
		table.WithFocused(true),
		table.WithHeight(15),
	)

	queueTable := table.New(
		table.WithFocused(true),
		table.WithHeight(15),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		Foreground(lipgloss.Color(config.Theme.Text)).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(config.Theme.Secondary)).
		BorderBottom(true).
		Bold(true)

	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color(config.Theme.Secondary)).
		Bold(true)

	playlistsTable.SetStyles(styles)
	songsTable.SetStyles(styles)
	queueTable.SetStyles(styles)

	return Model{
		log:             log,
		config:          config,
		db:              db,
		cacheRepository: cacheRepository,

		screen:         PlaylistScreen,
		previousScreen: PlaylistScreen,

		spinner:  s,
		progress: prog,

		currentTime: 20,
		duration:    100,
		isPlaying:   false,

		playlistsTable: playlistsTable,
		songsTable:     songsTable,
		queueTable:     queueTable,

		volume:        config.Player.Volume,
		favoritesOnly: config.General.ToggleFavorites,
	}
}

// Init initializes the model and fetches playlists from the cache repository.
func (m Model) Init() tea.Cmd {
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
