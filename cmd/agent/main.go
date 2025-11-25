package main

import (
	"log"
	agentconfig "metralert/config/agent"
	"metralert/internal/agent"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/context"
)

func main() {

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := agentconfig.Config{}
	cfg.GetConfig()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	log.Printf(`Запущен агент:
		ServerAddress %s,
		PollInterval: %d,
		ReportInterval: %d,
		RateLimit: %d`,
		cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, cfg.RateLimit)

	metricsAgent := agent.New(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, cfg.HashKey, sugar, true, cfg.CryptoKey)
	metricsAgent.StartSendPostWorkers(cfg.RateLimit)
	go metricsAgent.SendAllMetrics(ctx, metricsAgent.CollectRuntimeMetrics(), metricsAgent.CollectGopsutilMetrics(), metricsAgent.WorkerChanIn, metricsAgent.WorkerChanOut)

	<-shutdownCh
	cancel()
	sugar.Infow("Shutting down agent")
	time.Sleep(3 * time.Second)
}
