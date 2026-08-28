package mpv

import (
	"fmt"
	"os"
	"os/exec"
)

func StartMPV(volume int) (*exec.Cmd, error) {
	os.Remove("/tmp/ytmp")

	cmd := exec.Command(
		"mpv",
		"--idle=yes",
		fmt.Sprintf("--volume=%d", volume),
		"--input-ipc-server=/tmp/ytmp",
	)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}
