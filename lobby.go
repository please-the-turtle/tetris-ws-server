package main

import (
	"log/slog"

	"github.com/google/uuid"
)

type Lobby struct {
	ID         uuid.UUID
	Capacity   int
	server     *WsServer
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	close      chan bool
}

func NewLobby(maxClients int, ws *WsServer) *Lobby {
	if maxClients < 2 {
		slog.Warn("maxClients must be greate than 1")
		maxClients = 2
	}

	return &Lobby{
		ID:         uuid.New(),
		Capacity:   maxClients,
		server:     ws,
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
		close:      make(chan bool),
	}
}

func (l *Lobby) Run() {
	defer func() {
		l.closeLobby()
	}()

	for {
		select {
		case client := <-l.register:
			l.registerClient(client)
		case client := <-l.unregister:
			l.unregisterClient(client)
		case message := <-l.broadcast:
			l.broadcastMessage(message)
		case <-l.close:
			return
		}
	}
}

func (l *Lobby) closeLobby() {
	l.server.deleteLobby(l.ID)
	for _, c := range l.clients {
		if c.lobby != nil {
			c.lobby = nil
			c.lobby.unregister <- c
		}
	}
	close(l.register)
	close(l.unregister)
	close(l.broadcast)
	close(l.close)
}

func (l *Lobby) registerClient(c *Client) {
	if (len(l.clients) + 1) > l.Capacity {
		slog.Info("client was't registered")
		response := NewErrorResponse(l.ID.String(), "the lobby is crowded")
		c.send <- response.Encode()
	}

	l.clients[c.ID] = c
}

func (l *Lobby) unregisterClient(c *Client) {
	delete(l.clients, c.ID)

	if len(l.clients) == 0 {
		slog.Info("lobby removed", slog.Any("lobby", l))
		l.close <- true
	}
}

func (l *Lobby) broadcastMessage(m *Message) {
	for _, c := range l.clients {
		if c == m.Sender {
			continue
		}
		c.send <- m.Encode()
	}
}
