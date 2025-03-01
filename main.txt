package main

import (
	"metralert/cmd/agent"
	"metralert/cmd/server"
)

func main() {
	serverurl := "http://localhost:8080"

	server.NewServer(serverurl)
	a := agent.NewClient(serverurl)
	go agent.CollectMetric()
	go a.SendAllMetrics()

	select {}
}
