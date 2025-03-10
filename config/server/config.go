package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress string `env:"ADDRESS"`
}

func (cfg *Config) GetConfig() {
	err := env.Parse(&cfg)
	if err != nil {
		fmt.Println("Переменная окружения ADDRESS не определена")
	}

	if cfg.ServerAddress == "" {
		flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server url")
	}
	flag.Parse()
}
