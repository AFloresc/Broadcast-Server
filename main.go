package main

import (
	"broadcast-server/cmd"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		println("Uso: broadcast-server [start|connect]")
		return
	}
	switch os.Args[1] {
	case "start":
		cmd.StartServer()
	case "connect":
		cmd.ConnectClient()
	default:
		println("Comando desconocido:", os.Args[1])
	}
}
