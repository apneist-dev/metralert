package main

import (
	"log"
	agentconfig "metralert/config/agent"
	"metralert/internal/agent"
)

func main() {
	cfg := agentconfig.Config{}
	cfg.GetConfig()

	log.Printf(`Запущен агент:
		ServerAddress %s,
		PollInterval: %d,
		ReportInterval: %d`,
		cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)

	metrics_agent := agent.New(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)
	go metrics_agent.CollectMetric()
	go metrics_agent.SendAllMetrics()

	select {}
}
