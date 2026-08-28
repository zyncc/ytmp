package main

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/logger"
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

	log.Info("ytmp running")

	p := tea.NewProgram(
		ui.New(log, conf, db, cacheRepository),
	)
	if _, err := p.Run(); err != nil {
		log.Fatal("failed to start ytmp", zap.Error(err))
	}
}
