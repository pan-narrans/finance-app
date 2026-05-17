package config

import (
	"encoding/json"
	"log"
	"os"

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
		Sources:      make(map[string]string),
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
		log.Printf("Warning: Mappings file at %s is invalid. Starting with empty mappings. Error: %v", path, err)
		return data, nil
	}

	return data, nil
}

/*
WriteMappings saves the [domain.MappingData] to a JSON file.
*/
func WriteMappings(path string, data domain.MappingData) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, fileData, 0644)
}
