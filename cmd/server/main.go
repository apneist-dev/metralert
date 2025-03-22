package main

import (
	"metralert/internal/server"
	"metralert/internal/storage"

	serverconfig "metralert/config/server"

	"go.uber.org/zap"
)

func main() {
	cfg := serverconfig.Config{}
	cfg.GetConfig()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	storage := storage.New()

	server := server.New(cfg.ServerAddress, &storage, sugar)
	server.Start()
}
