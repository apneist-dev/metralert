package main

import (
	"log"
	agentconfig "metralert/config/agent"
	"metralert/internal/agent"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/net/context"
)

func main() {

	// shutdownCh := make(chan os.Signal, 1)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg := agentconfig.Config{}
	err = cfg.GetConfig()
	if err != nil {
		sugar.Fatalln("unable to read config file:", err)
	}

	log.Printf(`Запущен агент:
		ServerAddress %s,
		PollInterval: %d,
		ReportInterval: %d,
		RateLimit: %d`,
		cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, cfg.RateLimit)

	metricsAgent := agent.New(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, cfg.HashKey, sugar, true, cfg.CryptoKey)
	metricsAgent.StartSendPostWorkers(cfg.RateLimit)
	metricsAgent.SendAllMetrics(ctx, metricsAgent.CollectRuntimeMetrics(), metricsAgent.CollectGopsutilMetrics(), metricsAgent.WorkerChanIn, metricsAgent.WorkerChanOut)

}
