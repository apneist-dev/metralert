package main

import "metralert/internal/server"

func main() {
	serverurl := "http://localhost:8080"
	server.NewServer(serverurl)

}
