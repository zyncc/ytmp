package mpv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func StartMPV(volume int) (*exec.Cmd, error) {
	sockPath, err := SocketPath()
	if err != nil {
		return nil, err
	}

	if err := RemoveSocket(sockPath); err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"mpv",
		"--idle=yes",
		fmt.Sprintf("--volume=%d", volume),
		fmt.Sprintf("--input-ipc-server=%s", sockPath),
	)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

func SocketPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, "ytmp", "mpv.sock"), nil
}

func RemoveSocket(sockPath string) error {
	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove mpv socket %q: %w", sockPath, err)
	}

	return nil
}
