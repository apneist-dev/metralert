package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress   string `env:"ADDRESS" json:"address"`
	StoreInterval   int    `env:"STORE_INTERVAL" json:"store_interval"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"store_file"`
	Restore         bool   `env:"RESTORE" json:"restore"`
	DatabaseAddress string `env:"DATABASE_DSN" json:"database_dsn"`
	HashKey         string `env:"KEY" json:"-"`
	AuditFile       string `env:"AUDIT_FILE" json:"-"`
	AuditURL        string `env:"AUDIT_URL" json:"-"`
	CryptoKey       string `env:"CRYPTO_KEY" json:"crypto_key"`
	ConfigFile      string `env:"CONFIG" json:"-"`
}

func (cfg *Config) ParseFlags() {

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server url")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "file swap interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "metrics_database.json", "filename to store metrics")
	flag.BoolVar(&cfg.Restore, "r", true, "restore metrics on startup")
}

func (cfg *Config) GetConfig() {

	var fileCfg Config

	// flag config
	flag.StringVar(&cfg.ConfigFile, "c", "", "json config file")
	flag.StringVar(&cfg.ServerAddress, "a", "", "server url")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "file swap interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "metrics_database.json", "filename to store metrics")
	flag.BoolVar(&cfg.Restore, "r", false, "restore metrics on startup")
	flag.StringVar(&cfg.DatabaseAddress, "d", "", "database dsn")
	flag.StringVar(&cfg.HashKey, "k", "", "hash key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "path of a file to store audit logs")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "path of a file to store audit logs")
	flag.StringVar(&cfg.CryptoKey, "crypto_key", "", "private key")
	flag.Parse()

	// env config
	err := env.Parse(cfg)
	if err != nil {
		fmt.Println("не удалось выполнить парсинг переменных окружения")
	}

	// json config
	fmt.Println("reading config file: ", cfg.ConfigFile)
	if cfg.ConfigFile != "" {
		data, err := os.ReadFile(cfg.ConfigFile)
		if err != nil {
			fmt.Println("unable to read file", cfg.ConfigFile)
		}
		if err = UnmarshalJSON(data, &fileCfg); err != nil {
			fmt.Println("unable to unmarshal file", cfg.ConfigFile)
		}
	}

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = fileCfg.ServerAddress
	}
	if cfg.StoreInterval == 0 {
		cfg.StoreInterval = fileCfg.StoreInterval
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = fileCfg.FileStoragePath
	}
	if !cfg.Restore {
		cfg.Restore = fileCfg.Restore
	}
	if cfg.DatabaseAddress == "" {
		cfg.DatabaseAddress = fileCfg.DatabaseAddress
	}

	// set defaults
	if cfg.ServerAddress == "" {
		cfg.ServerAddress = "localhost:8080"
	}
	if cfg.StoreInterval == 0 {
		cfg.StoreInterval = 300
	}

	// _, StoreIntervalSet := os.LookupEnv("STORE_INTERVAL")
	// _, RestoreSet := os.LookupEnv("RESTORE")

}

func UnmarshalJSON(data []byte, c *Config) (err error) {
	type ConfigAlias Config

	aliasValue := &struct {
		*ConfigAlias
		StoreInterval string `json:"store_interval"`
	}{ConfigAlias: (*ConfigAlias)(c)}

	if err = json.Unmarshal(data, aliasValue); err != nil {
		return
	}
	aliasValue.StoreInterval = strings.TrimSuffix(aliasValue.StoreInterval, "s")
	c.StoreInterval, err = strconv.Atoi(aliasValue.StoreInterval)
	if err != nil {
		return err
	}
	return
}
