package ui

import (
	"context"
	"database/sql"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/mpv"
	"github.com/zyncc/ytmp/internal/queue"
	"go.uber.org/zap"
)

type Screen int

const (
	PlaylistScreen Screen = iota
	SongsScreen
	QueueScreen
	KeybindsScreen
)

type playlistsLoadedMsg struct {
	playlists []cache.Playlist
	err       error
}

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
	isPaused    bool
	isMuted     bool

	playlistsTable table.Model
	songsTable     table.Model
	queueTable     table.Model

	keybindsViewport viewport.Model

	playlists []cache.Playlist
	songs     []cache.Song

	fetchCancel context.CancelFunc

	urlCache          map[string]string
	currentPlayingURL string

	selectedPlaylist cache.Playlist

	q queue.Queue

	repeatMode bool

	volume                int
	volumeIncrementAmount int
	favoritesOnly         bool

	width  int
	height int

	mpvClient *mpv.Client
	mpvEvents <-chan mpv.Event
}

func New(log *zap.Logger, config *config.Config, db *sql.DB, cacheRepository cache.Storer, mpvClient *mpv.Client, mpvEvents <-chan mpv.Event) Model {
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

		currentTime: 0,
		duration:    100,
		isPlaying:   false,
		isPaused:    false,

		repeatMode: false,

		playlistsTable: playlistsTable,
		songsTable:     songsTable,
		queueTable:     queueTable,

		keybindsViewport: viewport.New(),

		volume:                config.Player.Volume,
		volumeIncrementAmount: config.Player.VolumeIncrementAmount,
		favoritesOnly:         config.General.ToggleFavorites,

		urlCache: make(map[string]string),

		mpvClient: mpvClient,
		mpvEvents: mpvEvents,
	}
}

func waitForMPVEvent(events <-chan mpv.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return nil
		}
		return ev
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForMPVEvent(m.mpvEvents),
		func() tea.Msg {
			playlists, err := m.cacheRepository.FetchPlaylists(m.config.General.ToggleFavorites)
			return playlistsLoadedMsg{
				playlists: playlists,
				err:       err,
			}
		},
	)
}
