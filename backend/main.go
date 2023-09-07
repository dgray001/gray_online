package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dgray001/gray_online/lobby"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	// Set environment variables
	for _, envvar := range environment_variables {
		os.Setenv(envvar.name, envvar.value)
	}

	// Run game lobby
	lobby_object := lobby.CreateLobby()
	go lobby_object.Run()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		api_lobby := api.Group("/lobby")
		{
			api_rooms := api_lobby.Group("/rooms")
			{
				api_rooms.POST("/get", func(c *gin.Context) {
					c.JSON(200, successResponse(lobby_object.GetRooms()))
				})
				api_rooms.GET("/create", func(c *gin.Context) {
					upgrader.CheckOrigin = func(r *http.Request) bool {
						return true
					}
					conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
					if err != nil {
						fmt.Println(err)
						return
					}
					lobby_object.AddRoom <- conn
				})
			}
		}
	}

	// demo websocket
	r.GET("/api/ws", func(c *gin.Context) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()
		for {
			conn.WriteMessage(websocket.TextMessage, []byte("Hello, WebSocket!"))
			time.Sleep(time.Second)
		}
	})

	r.Run()
}
