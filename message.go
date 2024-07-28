package main

const (
	SendMessageAction = "send-message"
	CreateLobbyAction = "create-lobby"
	JoinLobbyAction   = "join-lobby"
	LeaveLobbyAction  = "leave-lobby"
)

type Message struct {
	Action  string  `json:"action"`
	Content string  `json:"content"`
	Sender  *Client `json:"client"`
}

func (m *Message) Encode() []byte {
	return EncodeToJSON(m)
}
