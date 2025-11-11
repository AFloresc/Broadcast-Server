package cmd

import (
	"broadcast-server/internal/server"
	"log"
	"net/http"
)

func StartServer() {
	hub := server.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.ServeWs(hub, w, r)
	})

	log.Println("Servidor escuchando en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
