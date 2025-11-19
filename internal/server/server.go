package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients       map[*Client]bool
	aliasToClient map[string]*Client
	aliasHistory  map[string]string
	broadcast     chan []byte
	register      chan *Client
	unregister    chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		aliasToClient: make(map[string]*Client),
		aliasHistory:  make(map[string]string),
		broadcast:     make(chan []byte),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

type Client struct {
	conn  *websocket.Conn
	send  chan []byte
	alias string
	color string
}

var colorPool = []string{
	"\033[31m", "\033[32m", "\033[33m", "\033[34m", "\033[35m", "\033[36m",
}
var colorIndex = 0

var colorNames = map[string]string{
	"\033[31m": "red",
	"\033[32m": "green",
	"\033[33m": "yellow",
	"\033[34m": "blue",
	"\033[35m": "magenta",
	"\033[36m": "cyan",
}

var nameToColor = map[string]string{
	"red":     "\033[31m",
	"green":   "\033[32m",
	"yellow":  "\033[33m",
	"blue":    "\033[34m",
	"magenta": "\033[35m",
	"cyan":    "\033[36m",
}

func nextColor() string {
	c := colorPool[colorIndex%len(colorPool)]
	colorIndex++
	return c
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Recibir alias inicial
	_, aliasMsg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}
	alias := string(aliasMsg)

	// Asignar color único
	color := nextColor()

	client := &Client{
		conn:  conn,
		send:  make(chan []byte, 256),
		alias: alias,
		color: color,
	}

	hub.register <- client
	hub.aliasToClient[alias] = client

	/*
		announcement := fmt.Sprintf(`{"alias":"server","color":"\033[36m","text":"📢 %s se ha conectado"}`, alias)
		hub.broadcast <- []byte(announcement)
	*/
	count := len(hub.clients)
	status := fmt.Sprintf(`{"alias":"server","color":"\033[36m","text":"👥 Ahora hay %d clientes conectados"}`, count)
	hub.broadcast <- []byte(status)

	go client.writePump()
	go func() {
		defer func() {
			hub.unregister <- client
			client.conn.Close()
			delete(hub.aliasToClient, client.alias)
			/*
				announcement := fmt.Sprintf(`{"alias":"server","color":"\033[33m","text":"📴 %s se ha desconectado"}`, client.alias)
				hub.broadcast <- []byte(announcement)
			*/
			count := len(hub.clients) - 1 // aún no se ha eliminado
			status := fmt.Sprintf(`{"alias":"server","color":"\033[33m","text":"👥 Ahora hay %d clientes conectados"}`, count)
			hub.broadcast <- []byte(status)
		}()
		for {
			_, message, err := client.conn.ReadMessage()
			if err != nil {
				break
			}
			text := string(message)

			// Comando: cambiar alias
			if strings.HasPrefix(text, "/alias ") {
				newAlias := strings.TrimSpace(strings.TrimPrefix(text, "/alias "))
				if newAlias == "" || hub.aliasToClient[newAlias] != nil {
					client.send <- []byte(`{"alias":"server","color":"\033[31m","text":"Alias inválido o en uso"}`)
					continue
				}
				hub.aliasHistory[newAlias] = client.alias
				delete(hub.aliasToClient, client.alias)
				client.alias = newAlias
				hub.aliasToClient[newAlias] = client
				client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[32m","text":"Alias cambiado a %s"}`, newAlias))
				continue
			}

			// Comando: consultar alias anterior
			if strings.HasPrefix(text, "/whowas ") {
				query := strings.TrimSpace(strings.TrimPrefix(text, "/whowas "))
				prev, ok := hub.aliasHistory[query]
				if ok {
					client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[36m","text":"%s era %s"}`, query, prev))
				} else {
					client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[33m","text":"No hay historial para %s"}`, query))
				}
				continue
			}

			// Comando: listar alias
			if strings.HasPrefix(text, "/list") {
				lines := []string{"👥 Conectados:"}
				for c := range hub.clients {
					colored := fmt.Sprintf("%s%s%s", c.color, c.alias, "\033[0m")
					lines = append(lines, colored)
				}
				response := strings.Join(lines, "\\n")
				client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[36m","text":"%s"}`, response))
				continue
			}

			// Comando: el servidor responde con el alias actual cel cliente
			if strings.HasPrefix(text, "/whoami") {
				client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[36m","text":"🪪 Tu alias actual es %s"}`, client.alias))
				continue
			}

			//Comando: retorna el color actual del alias
			if strings.HasPrefix(text, "/color") {
				name := colorNames[client.color]
				if name == "" {
					name = "desconocido"
				}
				client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"%s","text":"🎨 Tu color actual es %s"}`, client.color, name))
				continue
			}

			// Comando: nuestra todos los colores disponibles
			if strings.HasPrefix(text, "/colors") {
				lines := []string{"🎨 Colores disponibles:"}
				for code, name := range colorNames {
					// Renderiza el nombre y el código en su color
					colored := fmt.Sprintf("%s%s%s: %s", code, name, "\033[0m", code)
					lines = append(lines, colored)
				}

				response := strings.Join(lines, "\\n")
				client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[36m","text":"%s"}`, response))
				continue
			}

			// Comando: cambnia tu color actual
			if strings.HasPrefix(text, "/color ") {
				requested := strings.TrimSpace(strings.TrimPrefix(text, "/color "))
				newColor, ok := nameToColor[requested]
				if !ok {
					client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"\033[31m","text":"Color '%s' no reconocido. Usa /colors para ver opciones."}`, requested))
					continue
				}

				client.color = newColor
				client.send <- []byte(fmt.Sprintf(`{"alias":"server","color":"%s","text":"🎨 Color cambiado a %s"}`, newColor, requested))

				announcement := fmt.Sprintf(`{"alias":"server","color":"%s","text":"🎨 %s ha cambiado su color a %s"}`, newColor, client.alias, requested)
				hub.broadcast <- []byte(announcement)
				continue
			}

			// Mensaje normal
			msg := fmt.Sprintf(`{"alias":"%s","color":"%s","text":"%s"}`, client.alias, client.color, text)
			hub.broadcast <- []byte(msg)
		}
	}()
}

func (c *Client) writePump() {
	for msg := range c.send {
		c.conn.WriteMessage(websocket.TextMessage, msg)
	}
}
