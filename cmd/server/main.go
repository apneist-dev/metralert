package main

import (
	"log"
	"metralert/internal/server"
	"metralert/internal/storage"

	serverconfig "metralert/config/server"
)

func main() {
	cfg := serverconfig.Config{}
	cfg.GetConfig()

	storage := storage.New()
	server := server.New(cfg.ServerAddress, &storage)
	log.Printf("Запущен сервер с адресом %s", cfg.ServerAddress)
	server.Start()
}
