package main

import (
	"fmt"
	"runtime"
)

func main() {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	fmt.Println(rtm.Alloc)
}
