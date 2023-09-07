package lobby

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	client_id    uint64
	nickname     string
	connection   *websocket.Conn
	send_message chan lobbyMessage
}

type lobbyMessage struct {
	Sender  string `json:"sender"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Data    string `json:"data"`
}

func CreateClient(connection *websocket.Conn, nickname string) *Client {
	client := Client{
		client_id:    0,
		nickname:     nickname,
		connection:   connection,
		send_message: make(chan lobbyMessage),
	}
	go client.run()
	return &client
}

func (c *Client) run() {
	for {
		select {
		case message := <-c.send_message:
			c.sendMessage(&message)
		}
	}
}

func (c *Client) sendMessage(message *lobbyMessage) {
	c.connection.WriteJSON(message)
}

func (c *Client) valid() bool {
	if c.client_id < 1 {
		return false
	}
	if c.connection == nil {
		return false
	}
	return true
}
