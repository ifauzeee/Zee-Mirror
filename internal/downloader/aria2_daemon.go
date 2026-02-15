package downloader

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Aria2Daemon struct {
	Cmd       *exec.Cmd
	ConfigDir string
}

func NewAria2Daemon(configDir string) *Aria2Daemon {
	return &Aria2Daemon{
		ConfigDir: configDir,
	}
}

func (d *Aria2Daemon) Start() error {
	slog.Info("Starting aria2c daemon...")

	sessionPath := filepath.Join(d.ConfigDir, "aria2.session")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		_ = os.WriteFile(sessionPath, []byte(""), 0600)
	}

	args := []string{
		"--enable-rpc",
		"--rpc-listen-all=false",
		"--rpc-listen-port=6800",
		"--input-file=" + sessionPath,
		"--save-session=" + sessionPath,
		"--save-session-interval=60",
		"--max-concurrent-downloads=10",
		"--check-certificate=false",
	}

	secret := os.Getenv("ARIA2_RPC_SECRET")
	if secret != "" {
		args = append(args, "--rpc-secret="+secret)
	}

	d.Cmd = exec.Command("aria2c", args...)
	d.Cmd.Stdout = os.Stdout
	d.Cmd.Stderr = os.Stderr

	if err := d.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to start aria2c: %v", err)
	}

	time.Sleep(2 * time.Second)
	slog.Info("aria2c daemon started successfully")

	return nil
}

func (d *Aria2Daemon) Stop() {
	if d.Cmd != nil && d.Cmd.Process != nil {
		slog.Info("Stopping aria2c daemon...")
		_ = d.Cmd.Process.Kill()
	}
}
