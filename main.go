package main

import (
	"fmt"
	"github.com/caeril/brelay/logging"
	"github.com/caeril/brelay/server"
)

func main() {

	fmt.Printf("Starting BRelay\n")
	logging.Init()
	server.Run()
	logging.DeInit()

}
