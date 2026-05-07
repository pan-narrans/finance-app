package config

import (
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config represents application configuration.
type Config struct {
	LedgerRoot      string  `env:"LEDGER_ROOT" envDefault:"."`
	ConfigRoot      string  `env:"CONFIG_ROOT" envDefault:"./config"`
	TelegramToken   string  `env:"TELEGRAM_TOKEN"`
	TelegramUserIDs []int64 `env:"TELEGRAM_USER_IDS" envSeparator:","`
}

// Load loads configuration from .env file and environment variables.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	config := &Config{}
	if err := env.Parse(config); err != nil {
		return nil, err
	}

	return config, nil
}
