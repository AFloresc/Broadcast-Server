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

	go client.writePump()
	go func() {
		defer func() {
			hub.unregister <- client
			client.conn.Close()
			delete(hub.aliasToClient, client.alias)
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
