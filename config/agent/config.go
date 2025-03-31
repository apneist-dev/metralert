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

	flag.Parse()
}
