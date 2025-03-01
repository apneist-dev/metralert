package main

import "metralert/internal/agent"

func main() {
	serverurl := "http://localhost:8080"

	client := agent.NewClient(serverurl)
	go agent.CollectMetric()
	go client.SendAllMetrics()

	select {}
}
