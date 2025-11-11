package client

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

func Run(url string) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("❌ Error de conexión:", err)
		return
	}
	defer conn.Close()

	// Manejo de interrupciones (Ctrl+C)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Lectura de mensajes entrantes
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("🔌 Conexión cerrada.")
				return
			}
			fmt.Println("📨", string(msg))
		}
	}()

	// Entrada de usuario
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("💬 Escribe mensajes para enviar. Ctrl+C para salir.")
	for {
		select {
		case <-interrupt:
			fmt.Println("\n👋 Cerrando cliente...")
			return
		default:
			if scanner.Scan() {
				text := scanner.Text()
				if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
					fmt.Println("❌ Error al enviar:", err)
					return
				}
			}
		}
	}
}
