package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/zyncc/ytmp/internal/cache"
	"github.com/zyncc/ytmp/internal/config"
	"github.com/zyncc/ytmp/internal/logger"
	"github.com/zyncc/ytmp/internal/mpris"
	"github.com/zyncc/ytmp/internal/mpv"
	"github.com/zyncc/ytmp/internal/ui"
	"go.uber.org/zap"
)

func main() {
	if handled, err := runCommand(os.Args[1:]); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}

	runPlayer()
}

func runCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}

	if len(args) != 2 || args[0] != "delete" || args[1] != "cache" {
		return true, fmt.Errorf("usage: ytmp delete cache")
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return true, fmt.Errorf("get user cache directory: %w", err)
	}

	databasePath := cache.DatabasePath(filepath.Join(cacheDir, "ytmp"))
	if err := os.Remove(databasePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("cache database does not exist: %s\n", databasePath)
			return true, nil
		}
		return true, fmt.Errorf("delete cache database: %w", err)
	}

	fmt.Printf("deleted cache database: %s\n", databasePath)
	return true, nil
}

func runPlayer() {
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
	defer func() {
		_ = mpvCmd.Process.Kill()
		_ = mpvCmd.Wait()
	}()

	sockPath, err := mpv.SocketPath()
	if err != nil {
		log.Fatal("failed to get mpv socket path", zap.Error(err))
	}
	defer func() {
		if err := mpv.RemoveSocket(sockPath); err != nil {
			log.Warn("failed to remove mpv socket", zap.Error(err))
		}
	}()

	mpvClient, err := mpv.Connect(sockPath)
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

	mprisServer, err := mpris.New(log)
	if err != nil {
		log.Warn("failed to initialize MPRIS server", zap.Error(err))
	} else {
		defer mprisServer.Close()
	}

	log.Info("ytmp running")

	p := tea.NewProgram(
		ui.New(log, conf, db, cacheRepository, mpvClient, mpvEvents, mprisServer),
	)

	if mprisServer != nil {
		mprisServer.SetSender(func(msg any) {
			p.Send(msg)
		})
	}

	if _, err := p.Run(); err != nil {
		log.Fatal("failed to start ytmp", zap.Error(err))
	}
}
