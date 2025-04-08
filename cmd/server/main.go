package main

import (
	"log"
	"metralert/internal/server"
	"metralert/internal/storage"
	"os"
	"os/signal"

	serverconfig "metralert/config/server"

	"go.uber.org/zap"
)

func main() {

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt)

	cfg := serverconfig.Config{}
	cfg.GetConfig()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	storage := storage.New(cfg.FileStoragePath, cfg.Restore, cfg.DatabaseAddress, sugar)

	go storage.BackupService(cfg.StoreInterval)
	server := server.New(cfg.ServerAddress, storage, sugar)
	go server.Start()

	<-shutdownCh
	server.Shutdown()
	storage.Shutdown()

}
