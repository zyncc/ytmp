package main

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/logger"
	"github.com/zyncc/ytmp/internal/mpv"
	"github.com/zyncc/ytmp/internal/ui"
	"go.uber.org/zap"
)

func main() {
	log := logger.New("production")

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal("failed to get user cache directory", zap.Error(err))
	}

	ytmpCacheDir := filepath.Join(cacheDir, "ytmp")
	if err := os.MkdirAll(ytmpCacheDir, 0o755); err != nil {
		log.Fatal("failed to create cache directory", zap.Error(err))
	}

	conf, err := config.Load()
	if err != nil {
		log.Error("failed to load config", zap.Error(err))
		conf = config.Default()
	}

	db, err := cache.OpenDatabase(ytmpCacheDir)
	if err != nil {
		log.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	if err := cache.Migrate(db); err != nil {
		log.Fatal("failed to migrate database", zap.Error(err))
	}

	cacheRepository := cache.NewCacheRepository(db)

	mpvCmd, err := mpv.StartMPV(conf.Player.Volume)
	if err != nil {
		log.Fatal("failed to initialize mpv command", zap.Error(err))
	}
	defer mpvCmd.Process.Kill()

	mpvClient, err := mpv.Connect("/tmp/ytmp")
	if err != nil {
		log.Fatal("failed to connect to mpv sock", zap.Error(err))
	}
	_ = mpvClient.ObserveProperty(1, "time-pos")
	_ = mpvClient.ObserveProperty(2, "pause")
	_ = mpvClient.ObserveProperty(3, "duration")

	mpvEvents := make(chan mpv.Event)
	go func() {
		defer close(mpvEvents)
		if err := mpvClient.Listen(mpvEvents); err != nil {
			log.Error("mpv listener stopped", zap.Error(err))
		}
	}()

	log.Info("ytmp running")

	p := tea.NewProgram(
		ui.New(log, conf, db, cacheRepository, mpvClient, mpvEvents),
	)
	if _, err := p.Run(); err != nil {
		log.Fatal("failed to start ytmp", zap.Error(err))
	}
}
