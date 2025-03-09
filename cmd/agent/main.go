package main

import (
	"log"
	agentconfig "metralert/config/agent"
	"metralert/internal/agent"
)

func main() {
	cfg := agentconfig.Config{}
	cfg.GetConfig()

	log.Printf("Запущен агент:\nServerAddress %s,\nPollInterval: %d,\nReportInterval: %d", cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)
	client := agent.NewClient(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)
	go agent.CollectMetric()
	go client.SendAllMetrics()

	select {}
}
