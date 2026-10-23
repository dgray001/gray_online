package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/lobby"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    0,
	WriteBufferSize:   0,
	EnableCompression: true,
	CheckOrigin:       func(r *http.Request) bool { return true },
}

var DEV = os.Getenv("GAE_SERVICE") == ""

func main() {
	debug := flag.Bool("debug", false, "log debug-level messages")
	flag.Parse()
	if *debug {
		util.DefaultStd.SetLevel(util.LevelDebug)
	}
	if err := risq.LoadConfig("game/games/risq/config"); err != nil {
		util.Errorf(util.DefaultErr, "loading risq config: %v", err)
		os.Exit(1)
	}
	// Set environment variables
	if DEV {
		for _, envvar := range environment_variables {
			os.Setenv(envvar.name, envvar.value)
		}
	}

	// Run game lobby
	lobby.DEV = DEV
	lobby_object := lobby.CreateLobby(util.DefaultStd, util.DefaultErr)
	go lobby_object.Run()

	if DEV {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	if !DEV {
		r.SetTrustedProxies(nil)
	}
	// Serve static html
	r.Use(static.Serve("/", static.LocalFile("static", false)))

	// All api groupings
	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, successResponse("pong"))
		})

		risqGame := func(c *gin.Context) *risq.GameRisq {
			game_id, err := strconv.ParseUint(c.Param("game_id"), 10, 64)
			if err != nil {
				c.JSON(200, failureResponse("Invalid game id"))
				return nil
			}
			game, ok := lobby_object.GetGame(game_id).(*risq.GameRisq)
			if !ok {
				c.JSON(200, failureResponse("Risq game doesn't exist"))
			}
			return game
		}

		api.GET("/risq/tech-tree", func(c *gin.Context) {
			config, _ := risq.ConfigFromDir("")
			c.JSON(200, successResponse(risq.AllBuildingConfigsToFrontend(config)))
		})

		api.GET("/risq/resources", func(c *gin.Context) {
			config, _ := risq.ConfigFromDir("")
			c.JSON(200, successResponse(risq.AllResourceConfigsToFrontend(config)))
		})

		api.GET("/risq/tech-tree/:game_id", func(c *gin.Context) {
			if game := risqGame(c); game != nil {
				c.JSON(200, successResponse(game.TechTreeToFrontend()))
			}
		})

		api.GET("/risq/resources/:game_id", func(c *gin.Context) {
			if game := risqGame(c); game != nil {
				c.JSON(200, successResponse(game.ResourceConfigsToFrontend()))
			}
		})

		api.GET("/risq/maps", func(c *gin.Context) {
			c.JSON(200, successResponse(risq.AllMapNamesToFrontend()))
		})

		api.GET("/risq/ai-configs", func(c *gin.Context) {
			c.JSON(200, successResponse(risq.AllAiNamesToFrontend()))
		})

		if DEV {
			registerRisqDevRoutes(api)
		} else {
			util.EnablePerfToStdout()
		}
		registerPerfRoutes(api)

		api_lobby := api.Group("/lobby")
		{
			api_lobby.GET("/connect/:nickname", func(c *gin.Context) {
				nickname := c.Param("nickname")
				if nickname == "" {
					util.Error(util.DefaultErr, "Must specify nickname when connecting")
					return
				}
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					util.Error(util.DefaultErr, err)
					return
				}
				client := lobby.CreateClient(conn, nickname, lobby_object)
				lobby_object.AddClient <- client
			})

			api_lobby.GET("/reconnect/:nickname/:client_id", func(c *gin.Context) {
				nickname := c.Param("nickname")
				if nickname == "" {
					util.Error(util.DefaultErr, "Must specify nickname when reconnecting")
					return
				}
				client_id_string := c.Param("client_id")
				client_id, err := strconv.Atoi(client_id_string)
				if err != nil || client_id < 1 {
					util.Error(util.DefaultErr, "Must specify proper client_id when reconnecting")
					return
				}
				old_client := lobby_object.GetClient(uint64(client_id))
				if old_client == nil {
					util.Error(util.DefaultErr, "Cannot find old client to reconnect to")
					return
				}
				nickname = old_client.GetNickname()
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					util.Error(util.DefaultErr, err)
					return
				}
				client := lobby.CreateClient(conn, nickname, lobby_object)
				lobby_object.ReconnectClient <- lobby.MakeClientId(client, uint64(client_id))
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
						util.Error(util.DefaultErr, err)
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
						util.Error(util.DefaultErr, err)
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
						util.Error(util.DefaultErr, err)
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
						Token     string `json:"token"`
						Viewer    string `json:"viewer" binding:"required"`
					}
					var req GetGameReq
					err = c.ShouldBindJSON(&req)
					if err != nil {
						util.Error(util.DefaultErr, err)
						c.JSON(200, failureResponse("Binding error"))
						return
					}
					if req.Viewer != "true" {
						client := lobby_object.GetClient(req.Client_id)
						if client == nil || !client.ValidToken(req.Token) {
							c.JSON(200, failureResponse("Invalid token"))
							return
						}
					}
					result, err := game.GetBase().RequestToFrontend(req.Client_id, req.Viewer == "true")
					if err != nil {
						util.Error(util.DefaultErr, err)
						c.JSON(200, failureResponse(err.Error()))
						return
					}
					c.JSON(200, successResponse(result))
				})
			}
		}
	}

	r.Run()
}
