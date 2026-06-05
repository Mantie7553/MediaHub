package ytdlp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

// ProgressFunc receives integer percent updates parsed from yt-dlp's stdout.
type ProgressFunc func(percent int)

/*
Function:	Download
Purpose:	Run yt-dlp to fetch a source URL, extract audio as mp3, and write

	it to the given output template, reporting progress as it goes

Params:
  - sourceURL: URL yt-dlp will download from
  - outputTemplate: yt-dlp -o template, including any per-job subdirs
  - onProgress: optional callback receiving integer percent updates
*/
func (c *YtdlpClient) Download(sourceURL, outputTemplate string, onProgress ProgressFunc) error {
	cmd := c.command(
		"-x",
		"--audio-format", "mp3",
		"--newline",
		"-o", outputTemplate,
		sourceURL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// yt-dlp emits lines like "[download]  12.3% of  4.5MiB at  1.2MiB/s ETA 00:08".
	// We pull the percent from the second whitespace-delimited field.
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if onProgress != nil && strings.Contains(line, "[download]") && strings.Contains(line, "%") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				percentStr := strings.TrimSuffix(fields[1], "%")
				if pct, err := strconv.ParseFloat(percentStr, 64); err == nil {
					onProgress(int(pct))
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	// Some lines may report 99% before completion; force a final 100 once the binary exits cleanly.
	if onProgress != nil {
		onProgress(100)
	}
	return nil
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
		results = append(results, w.toDomain())
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// command builds an exec.Cmd for the yt-dlp binary; isolated so tests can stub it.
func (c *YtdlpClient) command(args ...string) *exec.Cmd {
	return exec.Command(c.config.BinaryPath, args...)
}
