package lobby

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	client_id    uint64
	connection   *websocket.Conn
	send_message chan string
}

func createClient(connection *websocket.Conn) *Client {
	client := Client{
		connection:   connection,
		send_message: make(chan string),
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

func (c *Client) sendMessage(message *string) {
	c.connection.WriteMessage(websocket.TextMessage, []byte(*message))
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
