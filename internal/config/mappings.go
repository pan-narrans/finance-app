package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/a-perez/finance-app/internal/domain"
)

/*
LoadMappings loads the mappings from a JSON file directly into [domain.MappingData].

If the file is missing or invalid, it returns empty mappings and logs a warning.
*/
func LoadMappings(path string) (domain.MappingData, error) {
	data := domain.MappingData{
		Accounts:     make(map[string]string),
		Descriptions: make(map[string]string),
		Cards:        make(map[string]string),
		Prefixes:     make([]string, 0),
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Warning: Mappings file not found at %s. Starting with empty mappings.", path)
			return data, nil
		}
		return data, err
	}

	if err := json.Unmarshal(fileData, &data); err != nil {
		return data, fmt.Errorf("invalid mappings JSON at %s: %w", path, err)
	}

	return data, nil
}

/*
WriteMappings saves the [domain.MappingData] to a JSON file.
It uses an atomic write pattern (write to temp file, then rename) to prevent corruption.
*/
func WriteMappings(path string, data domain.MappingData) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 1. Create a temporary file in the same directory
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "mappings-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Cleanup if we fail

	// 2. Write data to temp file
	if _, err := tmpFile.Write(fileData); err != nil {
		tmpFile.Close()
		return err
	}

	// 3. Sync and Close
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// 4. Atomic Rename
	return os.Rename(tmpPath, path)
}
