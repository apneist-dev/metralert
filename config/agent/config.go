package agentconfig

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress  string `env:"ADDRESS" json:"address"`
	ReportInterval int    `env:"REPORT_INTERVAL" json:"report_interval"`
	PollInterval   int    `env:"POLL_INTERVAL" json:"poll_interval"`
	HashKey        string `env:"KEY" json:"-"`
	RateLimit      int    `env:"RATE_LIMIT" json:"-"`
	CryptoKey      string `env:"CRYPTO_KEY" json:"crypto_key"`
	ConfigFile     string `env:"CONFIG" json:"-"`
}

func (cfg *Config) GetConfig() {

	var fileCfg Config

	// flag config
	flag.StringVar(&cfg.ServerAddress, "a", "", "server url")
	flag.IntVar(&cfg.ReportInterval, "r", 0, "reportInterval")
	flag.IntVar(&cfg.PollInterval, "p", 0, "pollInterval")
	flag.StringVar(&cfg.HashKey, "k", "", "hash key")
	flag.IntVar(&cfg.RateLimit, "l", 0, "rate limit")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "Public Key")

	flag.Parse()

	// env config
	err := env.Parse(cfg)
	if err != nil {
		log.Println("не удалось распарсить переменные окружения")
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

	if cfg.ReportInterval == 0 {
		cfg.ReportInterval = fileCfg.ReportInterval
	}

	if cfg.PollInterval == 0 {
		cfg.PollInterval = fileCfg.PollInterval
	}

	if cfg.CryptoKey == "" {
		cfg.CryptoKey = fileCfg.CryptoKey
	}

	// set defaults
	if cfg.ServerAddress == "" {
		cfg.ServerAddress = "localhost:8080"
	}
	if cfg.ReportInterval == 0 {
		cfg.ReportInterval = 10
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2
	}
	if cfg.RateLimit == 0 {
		cfg.RateLimit = 5
	}

}

func UnmarshalJSON(data []byte, c *Config) (err error) {
	type ConfigAlias Config

	aliasValue := &struct {
		*ConfigAlias
		ReportInterval string `json:"report_interval"`
		PollInterval   string `json:"poll_interval"`
	}{ConfigAlias: (*ConfigAlias)(c)}

	if err = json.Unmarshal(data, aliasValue); err != nil {
		return
	}
	aliasValue.ReportInterval = strings.TrimSuffix(aliasValue.ReportInterval, "s")
	c.ReportInterval, err = strconv.Atoi(aliasValue.ReportInterval)
	if err != nil {
		return err
	}

	aliasValue.PollInterval = strings.TrimSuffix(aliasValue.PollInterval, "s")
	c.PollInterval, err = strconv.Atoi(aliasValue.PollInterval)
	if err != nil {
		return err
	}
	return
}
