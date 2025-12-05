package main

import (
	"context"
	"fmt"
	"log"
	"metralert/internal/server"
	"metralert/internal/storage"
	"os"
	"os/signal"
	"syscall"

	serverconfig "metralert/config/server"

	"go.uber.org/zap"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func PrintTags() {
	fmt.Printf(`
Build version: %s
Build date: %s
Build commit: %s
	`, buildVersion, buildDate, buildCommit)
}

func main() {

	PrintTags()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg := serverconfig.Config{}
	err = cfg.GetConfig()
	if err != nil {
		sugar.Fatalln("unable to get config :", err)
	}

	storage := storage.NewStorage(cfg.FileStoragePath, cfg.Restore, cfg.DatabaseAddress, sugar)

	sugar.Infow("Config applied",
		"cfg", cfg)
	go storage.BackupService(cfg.StoreInterval)
	server := server.New(cfg.ServerAddress, storage, cfg.HashKey, sugar, cfg.CryptoKey)
	go server.Start()
	go server.AuditLogger(cfg.AuditFile, cfg.AuditURL)

	<-ctx.Done()
	server.Shutdown()
	storage.Shutdown()

}
