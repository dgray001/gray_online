package lobby

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	read_wait  = 30 * time.Second
	read_limit = 5120 // bytes
	ping_time  = 250 * time.Millisecond
	write_wait = 10 * time.Second
)

type Client struct {
	client_id    uint64
	nickname     string
	connection   *websocket.Conn
	send_message chan lobbyMessage
	lobby        *Lobby
}

type lobbyMessage struct {
	Sender  string `json:"sender"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Data    string `json:"data"`
}

func CreateClient(connection *websocket.Conn, nickname string, lobby *Lobby) *Client {
	client := Client{
		client_id:    0,
		nickname:     nickname,
		connection:   connection,
		send_message: make(chan lobbyMessage),
		lobby:        lobby,
	}
	go client.readMessages()
	go client.writeMessages()
	return &client
}

// Close send message channel
func (c *Client) close() {
	close(c.send_message)
	c.connection.Close()
}

func (c *Client) readMessages() {
	defer func() {
		c.lobby.RemoveClient <- c
		c.connection.Close()
	}()
	c.connection.SetReadLimit(read_limit)
	c.connection.SetReadDeadline(time.Now().Add(read_wait))
	c.connection.SetPongHandler(func(string) error {
		c.connection.SetReadDeadline(time.Now().Add(read_wait))
		return nil
	})
	for {
		var message lobbyMessage
		err := c.connection.ReadJSON(&message)
		if err != nil {
			fmt.Println("Error at message reader:", err)
			break
		}
		c.lobby.broadcast <- message
	}
}

func (c *Client) writeMessages() {
	ticker := time.NewTicker(ping_time)
	defer func() {
		c.connection.Close()
		ticker.Stop()
	}()
	for {
		select {
		case message, ok := <-c.send_message:
			c.connection.SetWriteDeadline(time.Now().Add(write_wait))
			if !ok {
				c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.connection.WriteJSON(message)
			if err != nil {
				fmt.Println("Error at message writer:", err)
				break
			}
		case <-ticker.C:
			c.connection.SetWriteDeadline(time.Now().Add(write_wait))
			err := c.connection.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				fmt.Println("Error at ping writer:", err)
				break
			}
		}
	}
}

func (c *Client) valid() bool {
	if c == nil {
		return false
	}
	if c.client_id < 1 {
		return false
	}
	if c.connection == nil {
		return false
	}
	return true
}
