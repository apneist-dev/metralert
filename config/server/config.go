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
	HashKey         string `env:"KEY"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
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
	} else {
		flag.String("a", "localhost:8080", "server url")
	}
	if cfg.StoreInterval == 0 && !StoreIntervalSet {
		flag.IntVar(&cfg.StoreInterval, "i", 300, "file swap interval")
	} else {
		flag.Int("i", 300, "file swap interval")
	}
	if cfg.FileStoragePath == "" {
		flag.StringVar(&cfg.FileStoragePath, "f", "metrics_database.json", "filename to store metrics")
	} else {
		flag.String("f", "metrics_database.json", "filename to store metrics")
	}
	if !RestoreSet && !cfg.Restore {
		flag.BoolVar(&cfg.Restore, "r", true, "restore metrics on startup")
	} else {
		flag.Bool("r", true, "restore metrics on startup")
	}
	// "host=localhost user=postgres password=postgres dbname=postgres sslmode=disable"
	// "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	if cfg.DatabaseAddress == "" {
		flag.StringVar(&cfg.DatabaseAddress, "d", "", "database dsn")
	}

	if cfg.HashKey == "" {
		flag.StringVar(&cfg.HashKey, "k", "", "hash key")
	} else {
		flag.String("k", "", "hash key")
	}

	if cfg.AuditFile == "" {
		flag.StringVar(&cfg.AuditFile, "audit-file", "", "path of a file to store audit logs")
	} else {
		flag.String("audit-file", "", "path of a file to store audit logs")
	}

	if cfg.AuditURL == "" {
		flag.StringVar(&cfg.AuditURL, "audit-url", "", "path of a file to store audit logs")
	} else {
		flag.String("audit-url", "", "URL to store audit logs")
	}

	flag.Parse()
}
