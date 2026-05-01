package ytdlp

import (
	"context"
	"os/exec"

	"github.com/Mantie7553/MediaHub/backend/internal/config"
)

// Client invokes the yt-dlp binary via os/exec for both downloads and music search.
// Each call spawns a subprocess; concurrency is the caller's responsibility (one
// goroutine per download_jobs row in MediaHub's case).
type Client struct {
	binaryPath  string
	downloadDir string
}

func New(cfg config.YTDLPConfig) *Client {
	return &Client{
		binaryPath:  cfg.BinaryPath,
		downloadDir: cfg.DownloadDir,
	}
}

// ProgressFunc receives integer percent updates parsed from yt-dlp's stdout.
type ProgressFunc func(percent int)

// Download fetches the source URL into a subdirectory of the configured download root.
// Progress updates are emitted as the worker parses yt-dlp's progress lines; the final
// call delivers 100 once the binary exits cleanly.
func (c *Client) Download(ctx context.Context, sourceURL, subDir string, onProgress ProgressFunc) error {
	return nil
}

// SearchMusic performs a `ytmsearch:` query and returns the parsed candidates.
// Used by the music module's discovery flow when no MusicBrainz match is suitable.
func (c *Client) SearchMusic(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return nil, nil
}

// command builds an exec.Cmd for the yt-dlp binary; isolated so tests can stub it.
func (c *Client) command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, c.binaryPath, args...)
}
