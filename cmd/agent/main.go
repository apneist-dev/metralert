package main

import (
	"flag"
	"fmt"
	"log"
	"metralert/internal/agent"

	"github.com/caarlos0/env/v6"
)

func main() {
	type Config struct {
		ServerAddress  string `env:"ADDRESS"`
		ReportInterval int    `env:"REPORT_INTERVAL"`
		PollInterval   int    `env:"pollInterval"`
	}
	var (
		cfg            Config
		serverurl      *string
		reportInterval *int
		pollInterval   *int
	)

	err := env.Parse(&cfg)
	if err != nil {
		fmt.Println("Переменная окружения ADDRESS не определена")
	}
	if cfg.ServerAddress == "" {
		serverurl = flag.String("a", "localhost:8080", "server url")
	} else {
		serverurl = &cfg.ServerAddress
	}

	if cfg.ReportInterval == 0 {
		reportInterval = flag.Int("r", 10, "reportInterval")
	} else {
		reportInterval = &cfg.ReportInterval
	}

	if cfg.PollInterval == 0 {
		pollInterval = flag.Int("p", 10, "pollInterval")
	} else {
		pollInterval = &cfg.PollInterval
	}

	flag.Parse()

	log.Printf("Запущен сервер с адресом %s", *serverurl)
	client := agent.NewClient(*serverurl, *pollInterval, *reportInterval)
	go agent.CollectMetric()
	go client.SendAllMetrics()

	select {}
}
