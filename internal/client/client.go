package client

import (
	"bufio"
	"fmt"
	"os"

	"github.com/gorilla/websocket"
)

func Run(url string) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("Error de conexión:", err)
		return
	}
	defer conn.Close()

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Println("📨", string(msg))
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Escribe mensajes para enviar:")
	for scanner.Scan() {
		text := scanner.Text()
		conn.WriteMessage(websocket.TextMessage, []byte(text))
	}
}
