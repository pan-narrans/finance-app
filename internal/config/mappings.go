package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/domain"
)

type mappingsFile struct {
	domain.MappingData
	Accounts map[string]json.RawMessage `json:"accounts"`
}

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

	file := mappingsFile{MappingData: data}
	if err := json.Unmarshal(fileData, &file); err != nil {
		return data, fmt.Errorf("invalid mappings JSON at %s: %w", path, err)
	}

	data = file.MappingData
	seen := make(map[string]bool)
	for key, value := range file.Accounts {
		normalized := strings.ToUpper(key)
		if seen[normalized] {
			return data, fmt.Errorf("duplicate account mapping keyword %q", key)
		}
		seen[normalized] = true
		if bytes.HasPrefix(bytes.TrimSpace(value), []byte("{")) {
			var roles domain.AccountRoles
			decoder := json.NewDecoder(bytes.NewReader(value))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&roles); err != nil || strings.TrimSpace(roles.Source) == "" || strings.TrimSpace(roles.Target) == "" {
				return data, fmt.Errorf("account mapping %q requires nonempty source and target", key)
			}
			if data.AccountRoles == nil {
				data.AccountRoles = make(map[string]domain.AccountRoles)
			}
			data.AccountRoles[key] = roles
		} else {
			var account string
			if err := json.Unmarshal(value, &account); err != nil || strings.TrimSpace(account) == "" {
				return data, fmt.Errorf("account mapping %q requires a nonempty account name", key)
			}
			data.Accounts[key] = account
		}
	}

	return data, nil
}

/*
WriteMappings saves the [domain.MappingData] to a JSON file.
It uses an atomic write pattern (write to temp file, then rename) to prevent corruption.
*/
func WriteMappings(path string, data domain.MappingData) error {
	file := mappingsFile{MappingData: data, Accounts: make(map[string]json.RawMessage)}
	for key, account := range data.Accounts {
		file.Accounts[key], _ = json.Marshal(account)
	}
	for key, roles := range data.AccountRoles {
		file.Accounts[key], _ = json.Marshal(roles)
	}
	fileData, err := json.MarshalIndent(file, "", "  ")
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
