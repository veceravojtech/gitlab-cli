package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/gitlab-cli/internal/gitlab"
)

func TestMRCache_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		cache     *MRCache
		wantValid bool
	}{
		{"nil cache", nil, false},
		{"fresh cache", &MRCache{Timestamp: time.Now()}, true},
		{"stale cache", &MRCache{Timestamp: time.Now().Add(-31 * time.Second)}, false},
		{"exact TTL", &MRCache{Timestamp: time.Now().Add(-30 * time.Second)}, false},
		{"just under TTL", &MRCache{Timestamp: time.Now().Add(-29 * time.Second)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cache.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestLoadMRCache_NoCacheFile(t *testing.T) {
	// Use a temp directory that won't have a cache file
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = tmpDir
	defer func() { cacheDir = originalCacheDir }()

	cache, err := LoadMRCache()
	if err != nil {
		t.Errorf("LoadMRCache() error = %v, want nil", err)
	}
	if cache != nil {
		t.Errorf("LoadMRCache() = %v, want nil (cache miss)", cache)
	}
}

func TestLoadMRCache_ValidCache(t *testing.T) {
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = tmpDir
	defer func() { cacheDir = originalCacheDir }()

	// Create a valid cache file
	cachePath := filepath.Join(tmpDir, cacheFileName)
	cacheData := MRCache{
		Timestamp: time.Now(),
		MRs: []gitlab.MergeRequest{
			{ID: 123, IID: 1, Title: "Test MR"},
		},
	}
	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatalf("Failed to marshal test cache: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}

	cache, err := LoadMRCache()
	if err != nil {
		t.Errorf("LoadMRCache() error = %v, want nil", err)
	}
	if cache == nil {
		t.Fatal("LoadMRCache() = nil, want cache data")
	}
	if len(cache.MRs) != 1 {
		t.Errorf("LoadMRCache().MRs length = %d, want 1", len(cache.MRs))
	}
	if cache.MRs[0].ID != 123 {
		t.Errorf("LoadMRCache().MRs[0].ID = %d, want 123", cache.MRs[0].ID)
	}
}

func TestLoadMRCache_StaleCache(t *testing.T) {
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = tmpDir
	defer func() { cacheDir = originalCacheDir }()

	// Create a stale cache file (31 seconds old)
	cachePath := filepath.Join(tmpDir, cacheFileName)
	cacheData := MRCache{
		Timestamp: time.Now().Add(-31 * time.Second),
		MRs: []gitlab.MergeRequest{
			{ID: 456, IID: 2, Title: "Stale MR"},
		},
	}
	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatalf("Failed to marshal test cache: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}

	// LoadMRCache returns the cache data; IsValid() determines freshness
	cache, err := LoadMRCache()
	if err != nil {
		t.Errorf("LoadMRCache() error = %v, want nil", err)
	}
	if cache == nil {
		t.Fatal("LoadMRCache() = nil, want cache data (stale but loadable)")
	}
	if cache.IsValid() {
		t.Error("cache.IsValid() = true, want false (stale cache)")
	}
}

func TestLoadMRCache_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = tmpDir
	defer func() { cacheDir = originalCacheDir }()

	// Create a corrupted cache file
	cachePath := filepath.Join(tmpDir, cacheFileName)
	if err := os.WriteFile(cachePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted cache file: %v", err)
	}

	// Corrupted file should return nil, nil (graceful degradation)
	cache, err := LoadMRCache()
	if err != nil {
		t.Errorf("LoadMRCache() error = %v, want nil (graceful degradation)", err)
	}
	if cache != nil {
		t.Errorf("LoadMRCache() = %v, want nil for corrupted file", cache)
	}
}

func TestSaveMRCache_CreatesDirectoryAndFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = filepath.Join(tmpDir, "subdir") // Non-existent subdir
	defer func() { cacheDir = originalCacheDir }()

	mrs := []gitlab.MergeRequest{
		{ID: 789, IID: 3, Title: "New MR", ProjectID: 100},
	}

	err := SaveMRCache(mrs)
	if err != nil {
		t.Fatalf("SaveMRCache() error = %v, want nil", err)
	}

	// Verify file was created
	cachePath := filepath.Join(cacheDir, cacheFileName)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("SaveMRCache() did not create cache file")
	}

	// Verify content is valid JSON
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var loaded MRCache
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("SaveMRCache() wrote invalid JSON: %v", err)
	}
	if len(loaded.MRs) != 1 {
		t.Errorf("Saved cache has %d MRs, want 1", len(loaded.MRs))
	}
	if loaded.MRs[0].ID != 789 {
		t.Errorf("Saved cache MR ID = %d, want 789", loaded.MRs[0].ID)
	}
}

func TestSaveMRCache_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = tmpDir
	defer func() { cacheDir = originalCacheDir }()

	// Save some MRs
	mrs := []gitlab.MergeRequest{
		{ID: 111, IID: 10, Title: "First MR", ProjectID: 50},
		{ID: 222, IID: 20, Title: "Second MR", ProjectID: 60},
	}

	if err := SaveMRCache(mrs); err != nil {
		t.Fatalf("SaveMRCache() error = %v", err)
	}

	// Load them back
	cache, err := LoadMRCache()
	if err != nil {
		t.Fatalf("LoadMRCache() error = %v", err)
	}
	if cache == nil {
		t.Fatal("LoadMRCache() = nil after save")
	}

	// Verify data integrity
	if len(cache.MRs) != 2 {
		t.Errorf("Round-trip MR count = %d, want 2", len(cache.MRs))
	}
	if cache.MRs[0].ID != 111 || cache.MRs[1].ID != 222 {
		t.Error("Round-trip MR IDs don't match")
	}
	if !cache.IsValid() {
		t.Error("Freshly saved cache should be valid")
	}
}

func TestSaveMRCache_TimestampIsRecent(t *testing.T) {
	tmpDir := t.TempDir()
	originalCacheDir := cacheDir
	cacheDir = tmpDir
	defer func() { cacheDir = originalCacheDir }()

	before := time.Now()
	if err := SaveMRCache([]gitlab.MergeRequest{}); err != nil {
		t.Fatalf("SaveMRCache() error = %v", err)
	}
	after := time.Now()

	cache, err := LoadMRCache()
	if err != nil {
		t.Fatalf("LoadMRCache() error = %v", err)
	}
	if cache == nil {
		t.Fatal("LoadMRCache() = nil")
	}

	if cache.Timestamp.Before(before) || cache.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", cache.Timestamp, before, after)
	}
}
