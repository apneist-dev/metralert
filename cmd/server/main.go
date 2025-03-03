package main

import (
	"flag"
	"fmt"
	"log"
	"metralert/internal/server"

	"github.com/caarlos0/env/v6"
)

func main() {
	type Config struct {
		ServerAddress string `env:"ADDRESS"`
	}
	var (
		cfg       Config
		serverurl *string
	)

	err := env.Parse(&cfg)
	if err != nil {
		fmt.Println("Переменная окружения ADDRESS не определена")
	}

	if cfg.ServerAddress == "" {
		serverurl = flag.String("a", "localhost:8080", "server url")
		flag.Parse()
	} else {
		serverurl = &cfg.ServerAddress
	}

	log.Printf("Запущен сервер с адресом %s", *serverurl)
	server.NewServer(*serverurl)
}
