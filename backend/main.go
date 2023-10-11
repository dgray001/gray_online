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
	//r.SetTrustedProxies(nil)

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
				api_rooms.GET("/get", func(c *gin.Context) {
					c.JSON(200, successResponse(lobby_object.GetRooms()))
				})
				api_rooms.GET("/get/:id", func(c *gin.Context) {
					id_param := c.Param("id")
					room_id, err := strconv.Atoi(id_param)
					if err != nil {
						fmt.Println(err)
						c.JSON(200, failureResponse("Invalid room id"))
						return
					}
					room := lobby_object.GetRoom(uint64(room_id))
					if room == nil {
						c.JSON(200, failureResponse("Room doesn't exist"))
						return
					}
					c.JSON(200, successResponse(room.ToFrontend()))
				})
			}

			api_users := api_lobby.Group("/users")
			{
				api_users.GET("/get", func(c *gin.Context) {
					c.JSON(200, successResponse(lobby_object.GetUsers()))
				})
				api_users.GET("/get/:id", func(c *gin.Context) {
					id_param := c.Param("id")
					client_id, err := strconv.Atoi(id_param)
					if err != nil {
						fmt.Println(err)
						c.JSON(200, failureResponse("Invalid client id"))
						return
					}
					client := lobby_object.GetClient(uint64(client_id))
					if client == nil {
						c.JSON(200, failureResponse("Client doesn't exist"))
						return
					}
					c.JSON(200, successResponse(client.ToFrontend()))
				})
			}

			api_games := api_lobby.Group("/games")
			{
				api_games.GET("/get", func(c *gin.Context) {
					c.JSON(200, successResponse(lobby_object.GetGames()))
				})
				api_games.POST("/get/:id", func(c *gin.Context) {
					id_param := c.Param("id")
					game_id, err := strconv.Atoi(id_param)
					if err != nil {
						fmt.Println(err)
						c.JSON(200, failureResponse("Invalid game id"))
						return
					}
					game := lobby_object.GetGame(uint64(game_id))
					if game == nil {
						c.JSON(200, failureResponse("Game doesn't exist"))
						return
					}
					type GetGameReq struct {
						Client_id uint64 `json:"client_id" binding:"required"`
						Viewer    string `json:"viewer" binding:"required"`
					}
					var req GetGameReq
					err = c.ShouldBindJSON(&req)
					if err != nil {
						fmt.Println(err)
						c.JSON(200, failureResponse("Binding error"))
						return
					}
					c.JSON(200, successResponse(game.ToFrontend(req.Client_id, req.Viewer == "true")))
				})
			}
		}
	}

	r.Run()
}
