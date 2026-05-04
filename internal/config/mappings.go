package config

import (
	"encoding/json"
	"os"

	"github.com/a-perez/finance-app/internal/domain"
)

// LoadMappings loads the mappings from a JSON file directly into domain.MappingData.
func LoadMappings(path string) (domain.MappingData, error) {
	data := domain.MappingData{
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
