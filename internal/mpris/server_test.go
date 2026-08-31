package mpris

import (
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/zyncc/ytmp/internal/cache"
	"go.uber.org/zap"
)

func TestMakeTrackPath(t *testing.T) {
	tests := []struct {
		input    string
		expected dbus.ObjectPath
	}{
		{"", "/org/mpris/MediaPlayer2/TrackList/NoTrack"},
		{"dQw4w9WgXcQ", "/org/mpris/MediaPlayer2/Track/dQw4w9WgXcQ"},
		{"invalid-id_123", "/org/mpris/MediaPlayer2/Track/invalid_id_123"},
		{"song/with/slashes", "/org/mpris/MediaPlayer2/Track/song_with_slashes"},
		{"---", "/org/mpris/MediaPlayer2/Track/___"},
	}

	for _, tt := range tests {
		result := makeTrackPath(tt.input)
		if result != tt.expected {
			t.Errorf("makeTrackPath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
		if !result.IsValid() {
			t.Errorf("makeTrackPath(%q) produced invalid DBus ObjectPath: %q", tt.input, result)
		}
	}
}



func TestServerLifecycleAndDispatch(t *testing.T) {
	log := zap.NewNop()
	srv, err := New(log)
	if err != nil {
		t.Skipf("skipping live dbus test: %v", err)
	}
	defer srv.Close()

	var receivedMsg any
	srv.SetSender(func(msg any) {
		receivedMsg = msg
	})

	player := &playerIface{srv: srv}
	root := &rootIface{srv: srv}

	// Test Player methods
	if err := player.PlayPause(); err != nil {
		t.Errorf("PlayPause returned error: %v", err)
	}
	if _, ok := receivedMsg.(MsgPlayPause); !ok {
		t.Errorf("expected MsgPlayPause, got %T", receivedMsg)
	}

	if err := player.Play(); err != nil {
		t.Errorf("Play returned error: %v", err)
	}
	if _, ok := receivedMsg.(MsgPlay); !ok {
		t.Errorf("expected MsgPlay, got %T", receivedMsg)
	}

	if err := player.Pause(); err != nil {
		t.Errorf("Pause returned error: %v", err)
	}
	if _, ok := receivedMsg.(MsgPause); !ok {
		t.Errorf("expected MsgPause, got %T", receivedMsg)
	}

	if err := player.Stop(); err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
	if _, ok := receivedMsg.(MsgStop); !ok {
		t.Errorf("expected MsgStop, got %T", receivedMsg)
	}

	if err := player.Next(); err != nil {
		t.Errorf("Next returned error: %v", err)
	}
	if _, ok := receivedMsg.(MsgNext); !ok {
		t.Errorf("expected MsgNext, got %T", receivedMsg)
	}

	if err := player.Previous(); err != nil {
		t.Errorf("Previous returned error: %v", err)
	}
	if _, ok := receivedMsg.(MsgPrevious); !ok {
		t.Errorf("expected MsgPrevious, got %T", receivedMsg)
	}

	if err := player.Seek(5_000_000); err != nil {
		t.Errorf("Seek returned error: %v", err)
	}
	if msg, ok := receivedMsg.(MsgSeek); !ok || msg.OffsetSeconds != 5 {
		t.Errorf("expected MsgSeek(5), got %+v", receivedMsg)
	}

	if err := player.OpenUri("https://youtube.com/watch?v=123"); err != nil {
		t.Errorf("OpenUri returned error: %v", err)
	}
	if msg, ok := receivedMsg.(MsgOpenURI); !ok || msg.URI != "https://youtube.com/watch?v=123" {
		t.Errorf("expected MsgOpenURI, got %+v", receivedMsg)
	}

	// Test State updates
	testSong := cache.Song{
		ID:           "test_id",
		Title:        "Test Title",
		Channel:      "Test Channel",
		Duration:     210,
		URL:          "https://youtube.com/watch?v=test",
		ThumbnailURL: "https://example.com/thumb.jpg",
	}

	srv.UpdateSong(testSong, "Test Album")
	srv.UpdatePlaybackStatus(true, false)
	srv.UpdatePosition(42)
	srv.UpdateVolume(75)
	srv.UpdateRepeatMode(true)
	srv.UpdateShuffle(true)
	srv.UpdateCanGo(true, true)
	srv.EmitSeeked(42)

	// Test SetPosition with matching track
	if err := player.SetPosition(makeTrackPath("test_id"), 42_000_000); err != nil {
		t.Errorf("SetPosition returned error: %v", err)
	}
	if msg, ok := receivedMsg.(MsgSetPosition); !ok || msg.PositionSeconds != 42 {
		t.Errorf("expected MsgSetPosition(42), got %+v", receivedMsg)
	}

	// Verify properties via D-Bus Get
	statusVar, dbusErr := srv.props.Get("org.mpris.MediaPlayer2.Player", "PlaybackStatus")
	if dbusErr != nil {
		t.Fatalf("failed to get PlaybackStatus: %v", dbusErr)
	}
	if statusVar.Value().(string) != "Playing" {
		t.Errorf("expected PlaybackStatus=Playing, got %v", statusVar.Value())
	}

	canPauseVar, dbusErr := srv.props.Get("org.mpris.MediaPlayer2.Player", "CanPause")
	if dbusErr != nil {
		t.Fatalf("failed to get CanPause: %v", dbusErr)
	}
	if canPauseVar.Value().(bool) != true {
		t.Errorf("expected CanPause=true, got %v", canPauseVar.Value())
	}

	volVar, dbusErr := srv.props.Get("org.mpris.MediaPlayer2.Player", "Volume")
	if dbusErr != nil {
		t.Fatalf("failed to get Volume: %v", dbusErr)
	}
	if volVar.Value().(float64) != 0.75 {
		t.Errorf("expected Volume=0.75, got %v", volVar.Value())
	}

	loopVar, dbusErr := srv.props.Get("org.mpris.MediaPlayer2.Player", "LoopStatus")
	if dbusErr != nil {
		t.Fatalf("failed to get LoopStatus: %v", dbusErr)
	}
	if loopVar.Value().(string) != "Track" {
		t.Errorf("expected LoopStatus=Track, got %v", loopVar.Value())
	}

	shufVar, dbusErr := srv.props.Get("org.mpris.MediaPlayer2.Player", "Shuffle")
	if dbusErr != nil {
		t.Fatalf("failed to get Shuffle: %v", dbusErr)
	}
	if shufVar.Value().(bool) != true {
		t.Errorf("expected Shuffle=true, got %v", shufVar.Value())
	}

	metaVar, dbusErr := srv.props.Get("org.mpris.MediaPlayer2.Player", "Metadata")
	if dbusErr != nil {
		t.Fatalf("failed to get Metadata: %v", dbusErr)
	}
	metaMap := metaVar.Value().(map[string]dbus.Variant)
	if metaMap["xesam:title"].Value().(string) != "Test Title" {
		t.Errorf("expected title 'Test Title', got %v", metaMap["xesam:title"].Value())
	}
	if metaMap["xesam:album"].Value().(string) != "Test Album" {
		t.Errorf("expected album 'Test Album', got %v", metaMap["xesam:album"].Value())
	}

	// Test ClearMetadata
	srv.ClearMetadata()
	metaVar, dbusErr = srv.props.Get("org.mpris.MediaPlayer2.Player", "Metadata")
	if dbusErr != nil {
		t.Fatalf("failed to get Metadata after clear: %v", dbusErr)
	}
	metaMap = metaVar.Value().(map[string]dbus.Variant)
	if metaMap["mpris:trackid"].Value().(dbus.ObjectPath) != "/org/mpris/MediaPlayer2/TrackList/NoTrack" {
		t.Errorf("expected trackid NoTrack, got %v", metaMap["mpris:trackid"].Value())
	}

	// Test root interface
	if err := root.Raise(); err != nil {
		t.Errorf("Raise returned error: %v", err)
	}
}
