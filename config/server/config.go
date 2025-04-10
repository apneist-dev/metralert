package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress   string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseAddress string `env:"DATABASE_DSN"`
}

func (cfg *Config) GetConfig() {
	err := env.Parse(cfg)
	if err != nil {
		fmt.Println("Переменная окружения ADDRESS не определена")
	}
	_, StoreIntervalSet := os.LookupEnv("STORE_INTERVAL")
	_, RestoreSet := os.LookupEnv("RESTORE")

	if cfg.ServerAddress == "" {
		flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server url")
	}
	if cfg.StoreInterval == 0 && !StoreIntervalSet {
		flag.IntVar(&cfg.StoreInterval, "i", 300, "file swap interval")
	}
	if cfg.FileStoragePath == "" {
		flag.StringVar(&cfg.FileStoragePath, "f", "metrics_database.json", "filename to store metrics")
	}
	if !RestoreSet && !cfg.Restore {
		flag.BoolVar(&cfg.Restore, "r", true, "restore metrics on startup")
	}
	// "host=localhost user=postgres password=postgres dbname=postgres sslmode=disable"
	// "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	if cfg.DatabaseAddress == "" {
		flag.StringVar(&cfg.DatabaseAddress, "d", "", "database dsn")
	}

	flag.Parse()
}
