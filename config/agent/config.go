package agentconfig

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress  string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	HashKey        string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

func (cfg *Config) GetConfig() {
	err := env.Parse(cfg)

	if err != nil {
		log.Println("Переменная окружения ADDRESS не определена")
	}
	if cfg.ServerAddress == "" {
		flag.StringVar(&cfg.ServerAddress, "a", "http://localhost:8080", "server url")
	}

	if cfg.ReportInterval == 0 {
		flag.IntVar(&cfg.ReportInterval, "r", 10, "reportInterval")
	}

	if cfg.PollInterval == 0 {
		flag.IntVar(&cfg.PollInterval, "p", 2, "pollInterval")
	}

	if cfg.HashKey == "" {
		flag.StringVar(&cfg.HashKey, "k", "", "hash key")
	} else {
		flag.String("k", "", "hash key")
	}

	if cfg.RateLimit == 0 {
		flag.IntVar(&cfg.RateLimit, "l", 5, "rate limit")
	}

	if cfg.CryptoKey == "" {
		flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "Public Key")
	}

	flag.Parse()
}
