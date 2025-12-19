package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/user/gitlab-cli/internal/gitlab"
)

const (
	cacheTTL      = 30 * time.Second
	cacheFileName = "mr-cache.json"
)

// cacheDir is the directory where cache files are stored.
// Can be overridden in tests.
var cacheDir string

// MRCache holds cached MR list data with a timestamp for TTL validation.
type MRCache struct {
	Timestamp time.Time              `json:"timestamp"`
	MRs       []gitlab.MergeRequest  `json:"mrs"`
}

// IsValid returns true if the cache is not nil and within the TTL window.
func (c *MRCache) IsValid() bool {
	if c == nil {
		return false
	}
	return time.Since(c.Timestamp) < cacheTTL
}

// getCacheDir returns the cache directory path, creating default if not set.
func getCacheDir() (string, error) {
	if cacheDir != "" {
		return cacheDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".gitlab-cli"), nil
}

// getCachePath returns the full path to the cache file.
func getCachePath() (string, error) {
	dir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cacheFileName), nil
}

// LoadMRCache loads the MR cache from disk.
// Returns nil, nil if cache file doesn't exist (cache miss).
// Returns nil, nil if cache file is corrupted (graceful degradation).
// Returns the cache data otherwise.
func LoadMRCache() (*MRCache, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, fmt.Errorf("getting cache path: %w", err)
	}

	file, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss, not an error
		}
		return nil, fmt.Errorf("opening cache file: %w", err)
	}
	defer file.Close()

	var cache MRCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		// Corrupted file - return nil for graceful degradation
		return nil, nil
	}

	return &cache, nil
}

// NoCacheEnabled returns true if the --no-cache flag was set.
// Used by the resolution layer to bypass cache.
func NoCacheEnabled() bool {
	return noCacheFlag
}

// SaveMRCache saves the MR list to the cache file with the current timestamp.
func SaveMRCache(mrs []gitlab.MergeRequest) error {
	cachePath, err := getCachePath()
	if err != nil {
		return fmt.Errorf("getting cache path: %w", err)
	}

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	cache := MRCache{
		Timestamp: time.Now(),
		MRs:       mrs,
	}

	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("creating cache file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cache); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}
