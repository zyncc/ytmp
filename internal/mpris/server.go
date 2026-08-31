package mpris

import (
	"fmt"
	"math"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/zyncc/ytmp/internal/cache"
	"go.uber.org/zap"
)

type MsgPlayPause struct{}
type MsgPlay struct{}
type MsgPause struct{}
type MsgStop struct{}
type MsgNext struct{}
type MsgPrevious struct{}

type MsgSeek struct {
	OffsetSeconds int
}

type MsgSetPosition struct {
	PositionSeconds int
}

type MsgSetVolume struct {
	Volume int
}

type MsgSetLoopStatus struct {
	LoopStatus string
}

type MsgSetShuffle struct {
	Shuffle bool
}

type MsgOpenURI struct {
	URI string
}

type Server struct {
	conn   *dbus.Conn
	props  *prop.Properties
	log    *zap.Logger
	sender func(msg any)
	mu     sync.RWMutex

	currentSong   cache.Song
	album         string
	currentTime   int
	duration      int
	isPlaying     bool
	isPaused      bool
	volume        int
	repeatMode    bool
	canGoNext     bool
	canGoPrevious bool
}

type rootIface struct {
	srv *Server
}

type playerIface struct {
	srv *Server
}

func makeTrackPath(id string) dbus.ObjectPath {
	if id == "" {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")
	}
	var clean []rune
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			clean = append(clean, r)
		} else {
			clean = append(clean, '_')
		}
	}
	if len(clean) == 0 {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")
	}
	return dbus.ObjectPath("/org/mpris/MediaPlayer2/Track/" + string(clean))
}

func New(log *zap.Logger) (*Server, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	srv := &Server{
		conn:   conn,
		log:    log,
		volume: 100,
	}

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.ytmp", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		pidName := fmt.Sprintf("org.mpris.MediaPlayer2.ytmp.instance%d", os.Getpid())
		reply, err = conn.RequestName(pidName, dbus.NameFlagDoNotQueue)
		if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to request dbus name: %w", err)
		}
	}

	propsMap := prop.Map{
		"org.mpris.MediaPlayer2": {
			"CanQuit":             &prop.Prop{Value: true, Writable: false, Emit: prop.EmitConst},
			"Fullscreen":          &prop.Prop{Value: false, Writable: false, Emit: prop.EmitFalse},
			"CanSetFullscreen":    &prop.Prop{Value: false, Writable: false, Emit: prop.EmitConst},
			"CanRaise":            &prop.Prop{Value: false, Writable: false, Emit: prop.EmitConst},
			"HasTrackList":        &prop.Prop{Value: false, Writable: false, Emit: prop.EmitConst},
			"Identity":            &prop.Prop{Value: "ytmp", Writable: false, Emit: prop.EmitConst},
			"DesktopEntry":        &prop.Prop{Value: "ytmp", Writable: false, Emit: prop.EmitConst},
			"SupportedUriSchemes": &prop.Prop{Value: []string{"http", "https"}, Writable: false, Emit: prop.EmitConst},
			"SupportedMimeTypes":  &prop.Prop{Value: []string{}, Writable: false, Emit: prop.EmitConst},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": &prop.Prop{Value: "Stopped", Writable: false, Emit: prop.EmitTrue},
			"LoopStatus": &prop.Prop{
				Value:    "None",
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					if loop, ok := c.Value.(string); ok {
						if loop != "None" && loop != "Track" && loop != "Playlist" {
							return prop.ErrInvalidArg
						}
						srv.send(MsgSetLoopStatus{LoopStatus: loop})
						return nil
					}
					return prop.ErrInvalidArg
				},
			},
			"Rate": &prop.Prop{
				Value:    1.0,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					var rateFloat float64
					switch v := c.Value.(type) {
					case float64:
						rateFloat = v
					case float32:
						rateFloat = float64(v)
					case int:
						rateFloat = float64(v)
					case int64:
						rateFloat = float64(v)
					default:
						return prop.ErrInvalidArg
					}
					if rateFloat == 1.0 {
						return nil
					}
					return prop.ErrInvalidArg
				},
			},
			"Shuffle": &prop.Prop{
				Value:    false,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					if shuf, ok := c.Value.(bool); ok {
						srv.send(MsgSetShuffle{Shuffle: shuf})
						return nil
					}
					return prop.ErrInvalidArg
				},
			},
			"Metadata": &prop.Prop{
				Value: map[string]dbus.Variant{
					"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")),
				},
				Writable: false,
				Emit:     prop.EmitTrue,
			},
			"Volume": &prop.Prop{
				Value:    1.0,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					var volFloat float64
					switch v := c.Value.(type) {
					case float64:
						volFloat = v
					case float32:
						volFloat = float64(v)
					case int:
						volFloat = float64(v)
					case int64:
						volFloat = float64(v)
					default:
						return prop.ErrInvalidArg
					}
					if volFloat < 0.0 {
						volFloat = 0.0
					}
					if volFloat > 1.0 {
						volFloat = 1.0
					}
					volInt := int(math.Round(volFloat * 100))
					srv.send(MsgSetVolume{Volume: volInt})
					return nil
				},
			},
			"Position":      &prop.Prop{Value: int64(0), Writable: false, Emit: prop.EmitFalse},
			"MinimumRate":   &prop.Prop{Value: 1.0, Writable: false, Emit: prop.EmitConst},
			"MaximumRate":   &prop.Prop{Value: 1.0, Writable: false, Emit: prop.EmitConst},
			"CanGoNext":     &prop.Prop{Value: false, Writable: false, Emit: prop.EmitTrue},
			"CanGoPrevious": &prop.Prop{Value: false, Writable: false, Emit: prop.EmitTrue},
			"CanPlay":       &prop.Prop{Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPause":      &prop.Prop{Value: false, Writable: false, Emit: prop.EmitTrue},
			"CanSeek":       &prop.Prop{Value: false, Writable: false, Emit: prop.EmitTrue},
			"CanControl":    &prop.Prop{Value: true, Writable: false, Emit: prop.EmitConst},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsMap)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export properties: %w", err)
	}
	srv.props = props

	root := &rootIface{srv: srv}
	player := &playerIface{srv: srv}

	if err := conn.Export(root, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export org.mpris.MediaPlayer2: %w", err)
	}

	if err := conn.Export(player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to export org.mpris.MediaPlayer2.Player: %w", err)
	}

	node := &introspect.Node{
		Name: "/org/mpris/MediaPlayer2",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       "org.mpris.MediaPlayer2",
				Methods:    introspect.Methods(root),
				Properties: props.Introspection("org.mpris.MediaPlayer2"),
			},
			{
				Name:       "org.mpris.MediaPlayer2.Player",
				Methods:    introspect.Methods(player),
				Properties: props.Introspection("org.mpris.MediaPlayer2.Player"),
				Signals: []introspect.Signal{
					{
						Name: "Seeked",
						Args: []introspect.Arg{
							{Name: "Position", Type: "x", Direction: "out"},
						},
					},
				},
			},
		},
	}

	if err := conn.Export(introspect.NewIntrospectable(node), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable"); err != nil {
		log.Warn("failed to export introspectable", zap.Error(err))
	}

	return srv, nil
}

func (s *Server) SetSender(sender func(msg any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sender = sender
}

func (s *Server) send(msg any) {
	s.mu.RLock()
	sender := s.sender
	s.mu.RUnlock()
	if sender != nil {
		sender(msg)
	}
}

func (r *rootIface) Raise() *dbus.Error {
	return nil
}

func (r *rootIface) Quit() *dbus.Error {
	r.srv.send(tea.Quit())
	return nil
}

func (p *playerIface) Next() *dbus.Error {
	p.srv.send(MsgNext{})
	return nil
}

func (p *playerIface) Previous() *dbus.Error {
	p.srv.send(MsgPrevious{})
	return nil
}

func (p *playerIface) Pause() *dbus.Error {
	p.srv.send(MsgPause{})
	return nil
}

func (p *playerIface) PlayPause() *dbus.Error {
	p.srv.send(MsgPlayPause{})
	return nil
}

func (p *playerIface) Stop() *dbus.Error {
	p.srv.send(MsgStop{})
	return nil
}

func (p *playerIface) Play() *dbus.Error {
	p.srv.send(MsgPlay{})
	return nil
}

func (p *playerIface) Seek(offset int64) *dbus.Error {
	offsetSec := int(math.Round(float64(offset) / 1_000_000.0))
	if offsetSec == 0 && offset != 0 {
		if offset > 0 {
			offsetSec = 1
		} else {
			offsetSec = -1
		}
	}
	p.srv.send(MsgSeek{OffsetSeconds: offsetSec})
	return nil
}

func (p *playerIface) SetPosition(trackID dbus.ObjectPath, position int64) *dbus.Error {
	p.srv.mu.RLock()
	currentPath := makeTrackPath(p.srv.currentSong.ID)
	hasSong := p.srv.currentSong.ID != ""
	p.srv.mu.RUnlock()

	if hasSong && trackID != "" && trackID != "/org/mpris/MediaPlayer2/TrackList/NoTrack" && trackID != currentPath {
		return nil
	}

	posSec := int(position / 1_000_000)
	p.srv.send(MsgSetPosition{PositionSeconds: posSec})
	return nil
}

func (p *playerIface) OpenUri(uri string) *dbus.Error {
	if uri != "" {
		p.srv.send(MsgOpenURI{URI: uri})
	}
	return nil
}

func buildMetadata(song cache.Song, album string, durationSec int) map[string]dbus.Variant {
	metadata := map[string]dbus.Variant{
		"mpris:trackid":     dbus.MakeVariant(makeTrackPath(song.ID)),
		"mpris:length":      dbus.MakeVariant(int64(durationSec) * 1_000_000),
		"xesam:title":       dbus.MakeVariant(song.Title),
		"xesam:artist":      dbus.MakeVariant([]string{song.Channel}),
		"xesam:albumArtist": dbus.MakeVariant([]string{song.Channel}),
	}
	if song.ThumbnailURL != "" {
		metadata["mpris:artUrl"] = dbus.MakeVariant(song.ThumbnailURL)
	}
	if song.URL != "" {
		metadata["xesam:url"] = dbus.MakeVariant(song.URL)
	}
	if album != "" {
		metadata["xesam:album"] = dbus.MakeVariant(album)
	}
	return metadata
}

func (s *Server) UpdateSong(song cache.Song, album string) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.currentSong = song
	s.album = album
	s.duration = song.Duration
	s.currentTime = 0
	s.mu.Unlock()

	metadata := buildMetadata(song, album, song.Duration)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Metadata", metadata)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", int64(0))
}

func (s *Server) ClearMetadata() {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.currentSong = cache.Song{}
	s.album = ""
	s.duration = 0
	s.currentTime = 0
	s.mu.Unlock()

	metadata := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")),
	}
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Metadata", metadata)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", int64(0))
}

func (s *Server) UpdatePlaybackStatus(isPlaying bool, isPaused bool) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.isPlaying = isPlaying
	s.isPaused = isPaused
	s.mu.Unlock()

	status := "Stopped"
	if isPlaying {
		if isPaused {
			status = "Paused"
		} else {
			status = "Playing"
		}
	}
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", status)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "CanPause", isPlaying)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "CanSeek", isPlaying)
}

func (s *Server) UpdatePosition(seconds int) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.currentTime = seconds
	s.mu.Unlock()

	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", int64(seconds)*1_000_000)
}

func (s *Server) UpdateDuration(seconds int) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.duration = seconds
	song := s.currentSong
	album := s.album
	s.mu.Unlock()

	metadata := buildMetadata(song, album, seconds)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Metadata", metadata)
}

func (s *Server) UpdateVolume(volume int) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.volume = volume
	s.mu.Unlock()

	volFloat := float64(volume) / 100.0
	if volFloat < 0 {
		volFloat = 0
	} else if volFloat > 1.0 {
		volFloat = 1.0
	}
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Volume", volFloat)
}

func (s *Server) UpdateRepeatMode(repeatMode bool) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.repeatMode = repeatMode
	s.mu.Unlock()

	loopStatus := "None"
	if repeatMode {
		loopStatus = "Track"
	}
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "LoopStatus", loopStatus)
}

func (s *Server) UpdateShuffle(shuffle bool) {
	if s == nil || s.props == nil {
		return
	}
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "Shuffle", shuffle)
}

func (s *Server) UpdateCanGo(canGoNext bool, canGoPrevious bool) {
	if s == nil || s.props == nil {
		return
	}
	s.mu.Lock()
	s.canGoNext = canGoNext
	s.canGoPrevious = canGoPrevious
	s.mu.Unlock()

	s.props.SetMust("org.mpris.MediaPlayer2.Player", "CanGoNext", canGoNext)
	s.props.SetMust("org.mpris.MediaPlayer2.Player", "CanGoPrevious", canGoPrevious)
}

func (s *Server) EmitSeeked(seconds int) {
	if s == nil || s.conn == nil {
		return
	}
	posMicroseconds := int64(seconds) * 1_000_000
	_ = s.conn.Emit("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Seeked", posMicroseconds)
}

func (s *Server) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}
