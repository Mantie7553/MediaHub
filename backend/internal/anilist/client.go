package ytdlp

import (
	"os"
	"os/exec"
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
	Purpose:	Run yt-dlp to fetch a source URL into the given output template
	Params:
		- sourceURL: URL yt-dlp will download from
		- outputTemplate: yt-dlp -o template, including any per-job subdirs
		- onProgress: optional callback receiving integer percent updates
*/
func (c *YtdlpClient) Download(sourceURL, outputTemplate string, onProgress ProgressFunc) error {
	return nil
}

/*
	Function:	SearchMusic
	Purpose:	Run a ytmsearch query and return parsed candidates
	Params:
		- query: free-text search term
		- limit: number of results to return
*/
func (c *YtdlpClient) SearchMusic(query string, limit int) ([]SearchResult, error) {
	return nil, nil
}

// command builds an exec.Cmd for the yt-dlp binary; isolated so tests can stub it.
func (c *YtdlpClient) command(args ...string) *exec.Cmd {
	return exec.Command(c.config.BinaryPath, args...)
}
