package config

import (
	"os"
	"strconv"
	"time"
)

// Config aggregates configuration for all external service clients.
// Database and JWT settings live elsewhere; this package is scoped to outbound integrations.
type Config struct {
	Anilist     AnilistConfig
	MusicBrainz MusicBrainzConfig
	Plex        PlexConfig
	YTDLP       YTDLPConfig
}

type AnilistConfig struct {
	BaseURL string
	Timeout time.Duration
}

// MusicBrainzConfig holds settings for the MusicBrainz REST API.
// UserAgent is required by their etiquette guidelines and should identify the app and a contact.
type MusicBrainzConfig struct {
	BaseURL   string
	UserAgent string
	Timeout   time.Duration
}

// PlexConfig holds settings for talking to a local Plex Media Server.
// Token is the X-Plex-Token used to authenticate every request.
type PlexConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

// YTDLPConfig holds settings for invoking the yt-dlp binary.
// DownloadDir is the root output directory; per-job subdirs are appended at runtime.
type YTDLPConfig struct {
	BinaryPath  string
	DownloadDir string
}

// Load reads external-service configuration from environment variables, applying
// defaults where one exists. Missing required values are not fatal here; callers
// will surface configuration errors at request time.
func Load() *Config {
	return &Config{
		Anilist: AnilistConfig{
			BaseURL: envDefault("ANILIST_URL", "https://graphql.anilist.co"),
			Timeout: envDuration("ANILIST_TIMEOUT_SEC", 10*time.Second),
		},
		MusicBrainz: MusicBrainzConfig{
			BaseURL:   envDefault("MUSICBRAINZ_URL", "https://musicbrainz.org/ws/2"),
			UserAgent: os.Getenv("MUSICBRAINZ_USER_AGENT"),
			Timeout:   envDuration("MUSICBRAINZ_TIMEOUT_SEC", 10*time.Second),
		},
		Plex: PlexConfig{
			BaseURL: os.Getenv("PLEX_URL"),
			Token:   os.Getenv("PLEX_TOKEN"),
			Timeout: envDuration("PLEX_TIMEOUT_SEC", 15*time.Second),
		},
		YTDLP: YTDLPConfig{
			BinaryPath:  envDefault("YTDLP_PATH", "yt-dlp"),
			DownloadDir: os.Getenv("YTDLP_DOWNLOAD_DIR"),
		},
	}
}

// envDefault returns the env var if set, otherwise the fallback.
func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envDuration parses an integer-seconds env var into a Duration, falling back if unset or invalid.
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}
