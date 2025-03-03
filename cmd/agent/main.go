package main

import (
	"flag"
	"metralert/internal/agent"
)

func main() {
	// serverurl := "localhost:8080"
	serverurl := flag.String("a", "localhost:8080", "server url")
	reportInterval := flag.Int("r", 10, "reportInterval")
	pollInterval := flag.Int("p", 2, "pollInterval")
	flag.Parse()
	client := agent.NewClient(*serverurl, *pollInterval, *reportInterval)
	go agent.CollectMetric()
	go client.SendAllMetrics()

	select {}
}
