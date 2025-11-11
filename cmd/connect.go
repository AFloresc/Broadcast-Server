package cmd

import (
	"broadcast-server/internal/client"
)

func ConnectClient() {
	client.Run("ws://localhost:8080/ws")
}
