package main

import (
	"fmt"
	"metralert/internal/reset"
)

func main() {
	projectDir := "."

	err := reset.ParseGen(projectDir)

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", projectDir, err)
		return
	}
}
