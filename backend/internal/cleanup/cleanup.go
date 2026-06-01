package cleanup

import (
	"os"
	"path/filepath"
	"time"
)

func Start(mediaRoot string) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			cleanTempDir(mediaRoot)
		}
	}()
}

func cleanTempDir(mediaRoot string) {
	tempDir := filepath.Join(mediaRoot, "temp")

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > 48*time.Hour {
			os.RemoveAll(filepath.Join(tempDir, entry.Name()))
		}
	}
}
