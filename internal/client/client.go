package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

var colors = []string{
	"\033[31m", // rojo
	"\033[32m", // verde
	"\033[33m", // amarillo
	"\033[34m", // azul
	"\033[35m", // magenta
	"\033[36m", // cyan
}

const reset = "\033[0m"

func Run(url string) {
	fmt.Print("🆔 Ingresa tu alias: ")
	aliasScanner := bufio.NewScanner(os.Stdin)
	aliasScanner.Scan()
	alias := aliasScanner.Text()

	// Conexión WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("❌ Error de conexión:", err)
		return
	}
	defer conn.Close()

	// Enviar alias al servidor
	conn.WriteMessage(websocket.TextMessage, []byte(alias))

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
			var data struct {
				Alias string `json:"alias"`
				Color string `json:"color"`
				Text  string `json:"text"`
			}
			if err := json.Unmarshal(msg, &data); err != nil {
				fmt.Println(string(msg)) // fallback
				continue
			}
			fmt.Printf("%s[%s]%s %s\n", data.Color, data.Alias, reset, data.Text)
		}
	}()

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
				msg := fmt.Sprintf("[%s] %s", alias, text)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					fmt.Println("❌ Error al enviar:", err)
					return
				}
			}
		}
	}
}

func colorFor(alias string) string {
	sum := 0
	for _, c := range alias {
		sum += int(c)
	}
	return colors[sum%len(colors)]
}
