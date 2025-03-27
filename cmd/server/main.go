package main

import (
	"metralert/internal/server"
	"metralert/internal/storage"
	"os"
	"os/signal"

	serverconfig "metralert/config/server"

	"go.uber.org/zap"
)

func main() {

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	cfg := serverconfig.Config{}
	cfg.GetConfig()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	storage := storage.New(cfg.FileStoragePath, cfg.Restore, sugar)

	go storage.BackupService(cfg.StoreInterval, false)
	server := server.New(cfg.ServerAddress, storage, sugar)
	go server.Start()

	<-stop
	storage.BackupService(cfg.StoreInterval, true)

}
