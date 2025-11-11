package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
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

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	_, aliasMsg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}
	alias := string(aliasMsg)

	client := &Client{
		conn:  conn,
		send:  make(chan []byte, 256),
		alias: alias,
		color: nextColor(),
	}
	hub.register <- client

	go client.writePump()
	go client.readPump(hub)
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		msg := fmt.Sprintf(`{"alias":"%s","color":"%s","text":"%s"}`, c.alias, c.color, string(message))
		hub.broadcast <- []byte(msg)
	}
}

func (c *Client) writePump() {
	for msg := range c.send {
		c.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func nextColor() string {
	c := colorPool[colorIndex%len(colorPool)]
	colorIndex++
	return c
}
