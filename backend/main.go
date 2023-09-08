package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/dgray001/gray_online/lobby"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
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

	// All api groupings
	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		api_lobby := api.Group("/lobby")
		{
			api_lobby.GET("/connect/:nickname", func(c *gin.Context) {
				nickname := c.Param("nickname")
				if nickname == "" {
					fmt.Println("Must specify nickname when connecting")
					return
				}
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					fmt.Println(err)
					return
				}
				client := lobby.CreateClient(conn, nickname, lobby_object)
				lobby_object.AddClient <- client
			})

			api_rooms := api_lobby.Group("/rooms")
			{
				api_rooms.POST("/get", func(c *gin.Context) {
					c.JSON(200, successResponse(lobby_object.GetRooms()))
				})
				api_rooms.POST("/create/:client_id", func(c *gin.Context) {
					client_id_param := c.Param("client_id")
					client_id, err := strconv.Atoi(client_id_param)
					if err != nil || client_id < 1 {
						c.JSON(200, failureResponse("client id not specified"))
						return
					}
					c.JSON(200, failureResponse("not implemented"))
				})
			}

			api_users := api_lobby.Group("/users")
			{
				api_users.POST("/get", func(c *gin.Context) {
					c.JSON(200, successResponse(lobby_object.GetUsers()))
				})
			}
		}
	}

	r.Run()
}
