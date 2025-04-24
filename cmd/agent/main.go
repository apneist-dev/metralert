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
		ReportInterval: %d`,
		cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)

	metricsAgent := agent.New(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, sugar, true)
	go metricsAgent.CollectMetric()
	go metricsAgent.SendAllMetrics()

	select {}
}
