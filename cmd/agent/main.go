package main

import (
	"log"
	agentconfig "metralert/config/agent"
	"metralert/internal/agent"

	"go.uber.org/zap"
)

func main() {
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

	metricsAgent := agent.New(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, cfg.HashKey, sugar, false)
	metricsAgent.StartSendPostWorkers(cfg.RateLimit)
	go metricsAgent.SendAllMetrics(metricsAgent.CollectRuntimeMetrics(), metricsAgent.CollectGopsutilMetrics(), metricsAgent.WorkerChanIn, metricsAgent.WorkerChanOut)

	select {}
}
