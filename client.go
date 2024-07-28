package main

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Max wait time when writing message to peer
	writeWait = 10 * time.Second

	// Max time till next pong from peer
	pongWait = 60 * time.Second

	// Send ping interval, must be less then pong wait time
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000
)

var (
	newLine = []byte{'\n'}
)

type Client struct {
	ID       uuid.UUID `json:"id"`
	Username string
	conn     *websocket.Conn
	wsServer *WsServer
	lobby    *Lobby
	send     chan []byte
}

func NewClient(conn *websocket.Conn, ws *WsServer) *Client {
	return &Client{
		ID:       uuid.New(),
		Username: "username",
		conn:     conn,
		wsServer: ws,
		send:     make(chan []byte),
	}
}

func (c *Client) listen() {
	go c.startReadLoop()
	go c.startWriteLoop()
}

func (c *Client) startReadLoop() {
	defer func() {
		c.disconnect()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, jsonMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				slog.Error(err.Error())
			}
			break
		}

		c.handleNewMessage(jsonMessage)
	}
}

func (c *Client) handleNewMessage(msgJSON []byte) {
	var msg Message
	if err := json.Unmarshal(msgJSON, &msg); err != nil {
		slog.Error(err.Error(), slog.String("msgJson", string(msgJSON)))
		return
	}

	msg.Sender = c

	switch msg.Action {
	case SendMessageAction:
		if c.lobby == nil {
			return
		}
		c.lobby.broadcast <- &msg
	case CreateLobbyAction:
		c.handleCreateLobbyMessage()
	case JoinLobbyAction:
		c.handleJoinLobbyMessage(&msg)
	case LeaveLobbyAction:
		c.handleLeaveLobbyMessage()
	default:
		slog.Warn("unknown message action", slog.String("action", msg.Action))
	}
}

func (c *Client) handleCreateLobbyMessage() {
	lobby := c.wsServer.createLobby()
	response := NewResponse(ResponseStatusLobbyCreated, lobby.ID.String())
	c.send <- response.Encode()
	c.lobby = lobby
	lobby.register <- c
}

func (c *Client) handleJoinLobbyMessage(msg *Message) {
	if c.lobby != nil {
		slog.Info("the client is already in lobby", slog.Any("client", c))
		response := NewErrorResponse(c.lobby.ID.String(), "the client is already in lobby")
		c.send <- response.Encode()
		return
	}

	lobbyID, err := uuid.Parse(msg.Content)
	if err != nil {
		slog.Error("lobby id is not valid", slog.String("lobbyID", msg.Content))
		response := NewErrorResponse(msg.Content, "lobby id is not valid")
		c.send <- response.Encode()
		return
	}

	lobby, prs := c.wsServer.lobbies[lobbyID]
	if !prs {
		slog.Info("lobby not found", slog.String("lobbyID", lobbyID.String()))
		response := NewErrorResponse(lobbyID.String(), "lobby not found")
		c.send <- response.Encode()
		return
	}

	c.lobby = lobby
	lobby.register <- c
}

func (c *Client) handleLeaveLobbyMessage() {
	if c.lobby == nil {
		return
	}
	c.lobby.unregister <- c
	c.lobby = nil
}

func (c *Client) startWriteLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			for i := 0; i < len(c.send); i++ {
				w.Write(newLine)
				message = <-c.send
				w.Write(message)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Client) disconnect() {
	c.wsServer.unregister <- c
	if c.lobby != nil {
		c.lobby.unregister <- c
	}
	close(c.send)
	c.conn.Close()
}
