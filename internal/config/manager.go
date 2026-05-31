package config

import (
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/fsnotify/fsnotify"
)

/*
Manager coordinates live reloading of configuration and mappings.
It implements the [ports.ConfigurationUseCase] interface.
*/
type Manager struct {
	current      atomic.Pointer[ports.AppConfig]
	watcher      *fsnotify.Watcher
	configPath   string
	mappingsPath string
	constructor  ports.MappingServiceConstructor
	repo         ports.TransactionRepository
	mu           sync.Mutex
}

/*
NewManager initializes a new ConfigManager.
It performs an initial load of the configuration files and starts the directory watcher.
*/
func NewManager(configPath, mappingsPath string, constructor ports.MappingServiceConstructor) (*Manager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	m := &Manager{
		configPath:   configPath,
		mappingsPath: mappingsPath,
		watcher:      watcher,
		constructor:  constructor,
	}

	if err := m.Reload(); err != nil {
		watcher.Close()
		return nil, err
	}

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
*/
func (m *Manager) Get() *ports.AppConfig {
	return m.current.Load()
}

/*
Reload forces a re-read of all configuration files and updates the atomic pointer.
*/
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reload()
}

/*
SetRepository sets the transaction repository for dynamic account discovery.
It triggers a reload to fetch the initial list of accounts.
*/
func (m *Manager) SetRepository(repo ports.TransactionRepository) {
	m.mu.Lock()
	m.repo = repo
	m.mu.Unlock()
	_ = m.Reload()
}

/*
reload performs the actual loading logic without acquiring the lock.
Must be called from a method that already holds m.mu.
*/
func (m *Manager) reload() error {
	settings, err := LoadConfig(m.configPath)
	if err != nil {
		return err
	}

	mappingsData, err := LoadMappings(m.mappingsPath)
	if err != nil {
		return err
	}

	mappingService := m.constructor(mappingsData)

	// Fetch dynamic accounts if repo is available
	if m.repo != nil {
		accounts, err := m.repo.GetAccounts()
		if err == nil {
			mappingService.LoadAccounts(accounts)
		} else {
			log.Printf("Warning: Dynamic account discovery failed: %v", err)
		}
	}

	m.current.Store(&ports.AppConfig{
		Settings: settings,
		Mappings: mappingService,
	})

	return nil
}

/*
ReloadWithData manually updates the manager with provided settings and mappings.
Primarily used for testing.
*/
func (m *Manager) ReloadWithData(settings domain.Settings, mappings domain.MappingData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mappingService := m.constructor(mappings)
	m.current.Store(&ports.AppConfig{
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

/*
SaveMappings persists the provided [domain.MappingData] and reloads the manager.
*/
func (m *Manager) SaveMappings(data domain.MappingData) error {
	if err := WriteMappings(m.mappingsPath, data); err != nil {
		return err
	}
	return m.Reload()
}

/*
UpdateMapping provides a thread-safe way to modify and persist mappings.
It reloads the latest data from disk before applying the update.
*/
func (m *Manager) UpdateMapping(fn func(data *domain.MappingData)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := LoadMappings(m.mappingsPath)
	if err != nil {
		return err
	}

	fn(&data)

	if err := WriteMappings(m.mappingsPath, data); err != nil {
		return err
	}

	return m.reload()
}

/*
LearnMapping updates the mappings based on transaction overrides and persists them.
*/
func (m *Manager) LearnMapping(transaction domain.Transaction, targetOverride bool, sourceOverride bool, originalSource string) error {
	if !targetOverride && (!sourceOverride || originalSource == "") {
		return nil
	}

	return m.UpdateMapping(func(data *domain.MappingData) {
		data.Learn(transaction, targetOverride, sourceOverride, originalSource)
	})
}
