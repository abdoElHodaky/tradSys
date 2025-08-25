package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"
)

// PluginMetadata contains cached metadata about a plugin
type PluginMetadata struct {
	// File information
	FilePath     string
	FileSize     int64
	ModTime      time.Time
	Hash         string
	
	// Plugin information
	Info         *PluginInfo
	
	// Validation status
	Validated    bool
	ValidatedAt  time.Time
	ValidationErrors []string
	
	// Performance metrics
	LoadDuration time.Duration
	MemoryUsage  int64
}

// NewPluginMetadata creates a new plugin metadata object
func NewPluginMetadata(filePath string) (*PluginMetadata, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	
	// Calculate file hash
	hash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, err
	}
	
	return &PluginMetadata{
		FilePath:     filePath,
		FileSize:     fileInfo.Size(),
		ModTime:      fileInfo.ModTime(),
		Hash:         hash,
		Validated:    false,
		ValidationErrors: make([]string, 0),
	}, nil
}

// IsModified checks if the plugin file has been modified since the metadata was created
func (m *PluginMetadata) IsModified() (bool, error) {
	// Get current file info
	fileInfo, err := os.Stat(m.FilePath)
	if err != nil {
		return false, err
	}
	
	// Check if the file size or modification time has changed
	if fileInfo.Size() != m.FileSize || !fileInfo.ModTime().Equal(m.ModTime) {
		return true, nil
	}
	
	// Calculate current hash
	currentHash, err := calculateFileHash(m.FilePath)
	if err != nil {
		return false, err
	}
	
	// Compare hashes
	return currentHash != m.Hash, nil
}

// UpdateMetadata updates the metadata with current file information
func (m *PluginMetadata) UpdateMetadata() error {
	// Get file info
	fileInfo, err := os.Stat(m.FilePath)
	if err != nil {
		return err
	}
	
	// Calculate file hash
	hash, err := calculateFileHash(m.FilePath)
	if err != nil {
		return err
	}
	
	// Update metadata
	m.FileSize = fileInfo.Size()
	m.ModTime = fileInfo.ModTime()
	m.Hash = hash
	
	return nil
}

// calculateFileHash calculates the SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Create a new hash
	hash := sha256.New()
	
	// Copy the file data to the hash
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	// Get the hash as a hex string
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	
	return hashString, nil
}
