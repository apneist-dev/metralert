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

	metricsAgent := agent.New(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)
	go metricsAgent.CollectMetric()
	go metricsAgent.SendAllMetrics()

	select {}
}
