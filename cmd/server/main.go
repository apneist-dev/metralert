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
	sugar.Infow("Config", "cfg.ServerAddress", cfg.ServerAddress, "cfg.Restore", cfg.Restore, "cfg.FileStoragePath", cfg.FileStoragePath, "cfg.DatabaseAddress", cfg.DatabaseAddress, "cfg.StoreInterval", cfg.StoreInterval)

	go storage.BackupService(cfg.StoreInterval)
	server := server.New(cfg.ServerAddress, storage, sugar)
	go server.Start()

	<-shutdownCh
	server.Shutdown()
	storage.Shutdown()

}
