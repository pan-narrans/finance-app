package config

import (
	"encoding/json"
	"log"
	"os"
)

// MappingData holds the raw configuration for transaction mappings.
type MappingData struct {
	Accounts     map[string]string `json:"accounts"`
	Descriptions map[string]string `json:"descriptions"`
	Sources      map[string]string `json:"sources"`
	Cards        map[string]string `json:"cards"`
	Prefixes     []string          `json:"prefixes"`
}

/*
LoadMappings loads the mappings from a JSON file directly into [MappingData].

If the file is missing or invalid, it returns empty mappings and logs a warning.
*/
func LoadMappings(path string) (MappingData, error) {
	data := MappingData{
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
