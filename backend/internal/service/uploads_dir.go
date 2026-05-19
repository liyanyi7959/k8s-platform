package service

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveUploadsDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("UPLOADS_DIR")); v != "" {
		return v, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(wd, "uploads"), nil
	}
	exeDir := filepath.Dir(exe)
	tmp := filepath.Clean(os.TempDir())
	if tmp != "" && strings.Contains(strings.ToLower(exeDir), strings.ToLower(tmp)) {
		return filepath.Join(wd, "uploads"), nil
	}
	return filepath.Join(exeDir, "uploads"), nil
}
