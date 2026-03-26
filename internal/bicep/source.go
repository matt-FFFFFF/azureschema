package bicep

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// BicepTypesBase is the base URL for raw bicep-types-az files on GitHub.
	BicepTypesBase = "https://raw.githubusercontent.com/Azure/bicep-types-az/main/generated"

	// IndexMaxAge is how long the cached index.json is considered fresh.
	IndexMaxAge = 24 * time.Hour
)

// Source provides access to bicep-types-az data files.
type Source interface {
	// ReadIndex returns the contents of index.json.
	ReadIndex(ctx context.Context) ([]byte, error)

	// ReadTypesFile returns the contents of a types.json file at the given relative path.
	ReadTypesFile(ctx context.Context, relPath string) ([]byte, error)
}

// LocalSource reads bicep-types-az data from a local directory.
type LocalSource struct {
	Dir string // path to the "generated" directory
}

func (s *LocalSource) ReadIndex(ctx context.Context) ([]byte, error) {
	p := filepath.Join(s.Dir, "index.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading local index %s: %w", p, err)
	}
	return data, nil
}

func (s *LocalSource) ReadTypesFile(ctx context.Context, relPath string) ([]byte, error) {
	p := filepath.Join(s.Dir, relPath)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading local types file %s: %w", p, err)
	}
	return data, nil
}

// RemoteSource fetches bicep-types-az data from GitHub with local file caching.
type RemoteSource struct {
	CacheDir string
	Client   *http.Client
}

// NewRemoteSource creates a RemoteSource with the default cache directory.
func NewRemoteSource() *RemoteSource {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "azure-schema")

	return &RemoteSource{
		CacheDir: cacheDir,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *RemoteSource) ensureCacheDir() error {
	return os.MkdirAll(s.CacheDir, 0o755)
}

func (s *RemoteSource) ReadIndex(ctx context.Context) ([]byte, error) {
	if err := s.ensureCacheDir(); err != nil {
		return nil, err
	}

	cachePath := filepath.Join(s.CacheDir, "index.json")

	// Check if cached file exists and is fresh.
	if info, err := os.Stat(cachePath); err == nil {
		age := time.Since(info.ModTime())
		if age < IndexMaxAge {
			return os.ReadFile(cachePath)
		}
	}

	// Fetch from remote.
	fmt.Fprintln(os.Stderr, "Fetching resource type index (cached for 24h)...")
	url := BicepTypesBase + "/index.json"
	data, err := s.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("downloading index from %s: %w", url, err)
	}

	// Write to cache.
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("caching index: %w", err)
	}

	return data, nil
}

func (s *RemoteSource) ReadTypesFile(ctx context.Context, relPath string) ([]byte, error) {
	if err := s.ensureCacheDir(); err != nil {
		return nil, err
	}

	// Create a safe cache filename by replacing path separators.
	cacheKey := strings.ReplaceAll(relPath, "/", "_")
	cachePath := filepath.Join(s.CacheDir, cacheKey)

	// Types files are cached permanently (they're versioned by API version in path).
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	}

	// Fetch from remote.
	fmt.Fprintf(os.Stderr, "Fetching types from %s...\n", relPath)
	url := BicepTypesBase + "/" + relPath
	data, err := s.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}

	// Write to cache.
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("caching types file: %w", err)
	}

	return data, nil
}

func (s *RemoteSource) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
