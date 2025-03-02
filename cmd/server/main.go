package main

import (
	"metralert/internal/server"
)

func main() {
	serverurl := "localhost:8080"
	server.NewServer(serverurl)
}
