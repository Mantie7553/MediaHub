package ytdlp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

/*
BinaryPath: path to the yt-dlp executable
DownloadDir: root output directory; per-job subdirs are appended at runtime
*/
type YtdlpConfig struct {
	BinaryPath  string
	DownloadDir string
}

/*
config: BinaryPath and DownloadDir for invoking yt-dlp
*/
type YtdlpClient struct {
	config YtdlpConfig
}

/*
Function:	NewYtdlpClient
Purpose:	Build a wrapper around the yt-dlp binary
Params:
  - binEnvKey: environment variable key holding the yt-dlp binary path
  - dirEnvKey: environment variable key holding the download root
*/
func NewYtdlpClient(binEnvKey string, dirEnvKey string) *YtdlpClient {
	binaryPath := os.Getenv(binEnvKey)
	// Fall back to the binary's name on PATH if no override was supplied.
	if binaryPath == "" {
		binaryPath = "yt-dlp"
	}
	return &YtdlpClient{
		config: YtdlpConfig{
			BinaryPath:  binaryPath,
			DownloadDir: os.Getenv(dirEnvKey),
		},
	}
}

/*
Function:	SearchMusic
Purpose:	Run a YouTube Music search via yt-dlp's ytmsearch prefix and return
parsed candidates without downloading anything
Params:
  - query: free-text search term
  - limit: number of results to return; defaults to 10 when zero or negative
*/
func (c *YtdlpClient) SearchMusic(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// ytmsearchN: tells yt-dlp to query YouTube Music for up to N results.
	searchTerm := fmt.Sprintf("ytsearch%d:%s", limit, query)

	cmd := c.command(
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
		"--quiet",
		searchTerm,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	results := []SearchResult{}
	scanner := bufio.NewScanner(stdout)
	// Per-result JSON can be very large (every available format, full description, etc.);
	// the default 64KB scanner buffer overflows on many videos, so bump it to 1MB.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var w wireSearchResult
		// Skip lines that aren't valid JSON rather than aborting the whole search.
		if err := json.Unmarshal(line, &w); err != nil {
			continue
		}
		if w.Duration > 30 {
			results = append(results, w.toDomain())
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// command builds an exec.Cmd for the yt-dlp binary; isolated so tests can stub it.
func (c *YtdlpClient) command(args ...string) *exec.Cmd {
	cmd := exec.Command(c.config.BinaryPath, args...)
	cmd.Env = Env()
	return cmd
}

/*
Function:	Env
Purpose:	the environment to run yt-dlp with. The release binary unpacks itself into TMPDIR on
every run, and the container points TMPDIR at the media disk (often a NAS) for transcodes,
which made each search wait on dozens of network file writes. YTDLP_TMPDIR, when set, gives
yt-dlp a local temp folder instead
*/
func Env() []string {
	env := os.Environ()
	dir := os.Getenv("YTDLP_TMPDIR")
	if dir == "" {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if !strings.HasPrefix(kv, "TMPDIR=") {
			out = append(out, kv)
		}
	}
	return append(out, "TMPDIR="+dir)
}
