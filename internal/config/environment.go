package config

import (
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Environment represents application configuration.
type Environment struct {
	LedgerRoot      string  `env:"LEDGER_ROOT" envDefault:"./sample-data/ledger-files"`
	LedgerFile      string  `env:"LEDGER_FILE" envDefault:"main.ledger"`
	ConfigRoot      string  `env:"CONFIG_ROOT" envDefault:"./config"`
	TelegramToken   string  `env:"TELEGRAM_TOKEN"`
	TelegramUserIDs []int64 `env:"TELEGRAM_USER_IDS" envSeparator:","`
	WebAppBaseURL   string  `env:"WEBAPP_BASE_URL" envDefault:"http://localhost:8080"`
	HTTPPort        int     `env:"HTTP_PORT" envDefault:"8080"`
}

// LoadEnvironment loads configuration from .env file.
func LoadEnvironment() (*Environment, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	config := &Environment{}
	if err := env.Parse(config); err != nil {
		return nil, err
	}

	return config, nil
}
