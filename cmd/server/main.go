package main

import (
	"flag"
	"metralert/internal/server"
)

func main() {
	serverurl := flag.String("a", "localhost:8080", "server url")
	flag.Parse()
	// serverurl := "localhost:8080"
	server.NewServer(*serverurl)
}
