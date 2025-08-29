package main

import (
	"os/exec"
	"path/filepath"
)

func runHdiffz(oldPath, newPath, outDiff string) error {
	args := []string{"-s-64", "-SD", "-c-zstd-21-24", "-d", oldPath, newPath, outDiff}
	hdiffzPath, err := filepath.Abs(filepath.Join("bin", "hdiffz.exe"))
	if err != nil {
		return err
	}
	cmd := exec.Command(hdiffzPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}
