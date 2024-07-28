package main

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const lobbyCapacity = 2

var upgrager = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type WsServer struct {
	clients    map[uuid.UUID]*Client
	lobbies    map[uuid.UUID]*Lobby
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func NewWebsocketServer() *WsServer {
	return &WsServer{
		clients:    make(map[uuid.UUID]*Client),
		lobbies:    make(map[uuid.UUID]*Lobby),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

func (s *WsServer) Run() {
	for {
		select {
		case client := <-s.register:
			s.registerClient(client)
		case client := <-s.unregister:
			s.unregisterClient(client)
		case message := <-s.broadcast:
			s.broadcastToClients(message)
		}
	}
}

func (s *WsServer) createLobby() *Lobby {
	l := NewLobby(lobbyCapacity, s)
	go l.Run()
	s.lobbies[l.ID] = l
	slog.Info("new lobby created", slog.Any("lobby", l))

	return l
}

func (s *WsServer) deleteLobby(lobbyId uuid.UUID) {
	delete(s.lobbies, lobbyId)
}

func (s *WsServer) broadcastToClients(message []byte) {
	for _, c := range s.clients {
		c.send <- message
	}
}

func (s *WsServer) registerClient(c *Client) {
	s.clients[c.ID] = c
	slog.Info("register new client", slog.Any("client", c))
}

func (s *WsServer) unregisterClient(c *Client) {
	delete(s.clients, c.ID)
	slog.Info("unregister client", slog.Any("client", c))
}

func (s *WsServer) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrager.Upgrade(w, r, nil)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	client := NewClient(conn, s)
	username := r.URL.Query().Get("username")
	if username != "" {
		client.Username = username
	}

	s.register <- client
	client.listen()

	slog.Info("new client connected", slog.Any("client", client))
}
