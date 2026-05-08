package config

import (
	"encoding/json"
	"os"
)

// MappingData holds the raw configuration for transaction mappings.
type MappingData struct {
	Accounts     map[string]string `json:"accounts"`
	Descriptions map[string]string `json:"descriptions"`
	Cards        map[string]string `json:"cards"`
	Prefixes     []string          `json:"prefixes"`
}

// LoadMappings loads the mappings from a JSON file directly into [MappingData].
func LoadMappings(path string) (MappingData, error) {
	data := MappingData{
		Accounts:     make(map[string]string),
		Descriptions: make(map[string]string),
		Cards:        make(map[string]string),
		Prefixes:     make([]string, 0),
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return data, err
	}

	if err := json.Unmarshal(fileData, &data); err != nil {
		return data, err
	}

	return data, nil
}
