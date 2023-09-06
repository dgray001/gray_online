package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

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

	r := gin.Default()
	r.SetTrustedProxies(nil)

	// demo get request
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// demo group
	v2 := r.Group("/api/group")
	{
		v2.GET("/one", one)
		v2.GET("/two", two)
		v2.GET("/three", three)
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

func one(c *gin.Context) {
	c.JSON(201, gin.H{
		"message": "1",
	})
}

func two(c *gin.Context) {
	c.JSON(202, gin.H{
		"message": "2",
	})
}

func three(c *gin.Context) {
	c.JSON(203, gin.H{
		"message": "3",
	})
}
