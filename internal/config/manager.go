package config

import (
	"log"
	"path/filepath"
	"sync/atomic"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/fsnotify/fsnotify"
)

/*
AppConfig combines application settings and the derived mapping service.
It represents a single, consistent snapshot of the application configuration.
*/
type AppConfig struct {
	Settings Config //TODO review naming
	Mappings ports.MappingProvider
}

/*
Manager coordinates live reloading of configuration and mappings.
It provides thread-safe access to the current configuration via an atomic pointer.
*/
type Manager struct {
	current      atomic.Pointer[AppConfig]
	watcher      *fsnotify.Watcher
	configPath   string
	mappingsPath string
	constructor  ports.MappingServiceConstructor
}

/*
NewManager initializes a new ConfigManager.
It performs an initial load of the configuration files and starts the directory watcher.
*/
func NewManager(configPath, mappingsPath string, constructor ports.MappingServiceConstructor) (*Manager, error) {
	m := &Manager{
		configPath:   configPath,
		mappingsPath: mappingsPath,
		constructor:  constructor,
	}

	// Initial load
	if err := m.Reload(); err != nil {
		return nil, err
	}

	// Set up watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	m.watcher = watcher

	// Watch the parent directory of config files
	configDir := filepath.Dir(configPath)
	if err := watcher.Add(configDir); err != nil {
		watcher.Close()
		return nil, err
	}

	return m, nil
}

/*
Get returns the current application configuration.
The returned pointer is safe to read but should not be modified.
*/
func (m *Manager) Get() *AppConfig {
	return m.current.Load()
}

/*
Reload forces a re-read of all configuration files and updates the atomic pointer.
*/
func (m *Manager) Reload() error {
	settings, err := LoadConfig(m.configPath)
	if err != nil {
		return err
	}

	mappingsData, err := LoadMappings(m.mappingsPath)
	if err != nil {
		return err
	}

	mappingService := m.constructor(mappingsData)

	m.current.Store(&AppConfig{
		Settings: settings,
		Mappings: mappingService,
	})

	return nil
}

/*
ReloadWithData manually updates the manager with provided settings and mappings.
Primarily used for testing.
*/
func (m *Manager) ReloadWithData(settings Config, mappings domain.MappingData) {
	mappingService := m.constructor(mappings)
	m.current.Store(&AppConfig{
		Settings: settings,
		Mappings: mappingService,
	})
}

/*
Watch starts the background goroutine that listens for file system events.
It should be called once after initializing the manager.
*/
func (m *Manager) Watch() {
	go func() {
		for {
			select {
			case event, ok := <-m.watcher.Events:
				if !ok {
					return
				}
				// We only care about changes to our specific config files
				if m.isRelevantFile(event.Name) && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) {
					log.Printf("Configuration change detected in %s. Reloading...", event.Name)
					if err := m.Reload(); err != nil {
						log.Printf("Error reloading configuration: %v", err)
					}
				}
			case err, ok := <-m.watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()
}

/*
Close stops the file system watcher.
*/
func (m *Manager) Close() error {
	return m.watcher.Close()
}

func (m *Manager) isRelevantFile(name string) bool {
	cleanName := filepath.Clean(name)
	return cleanName == filepath.Clean(m.configPath) || cleanName == filepath.Clean(m.mappingsPath)
}
