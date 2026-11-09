package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestConnectionIdentityPersistsCookie(t *testing.T) {
	old := DEV
	DEV = false
	defer func() { DEV = old }()
	r := gin.New()
	r.Use(connectionIdentity)
	r.GET("/connect/:nickname", func(c *gin.Context) { c.JSON(200, c.MustGet("identity")) })
	first := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/connect/test", nil)
	request.Header.Set("X-Forwarded-For", "spoofed, 203.0.113.1, 10.0.0.1")
	r.ServeHTTP(first, request)
	cookie := first.Result().Cookies()[0]
	if !cookie.Secure || !cookie.HttpOnly || len(cookie.Value) != 32 {
		t.Fatal("invalid identity cookie", cookie)
	}
	request = httptest.NewRequest("GET", "/connect/test", nil)
	request.AddCookie(cookie)
	second := httptest.NewRecorder()
	r.ServeHTTP(second, request)
	if second.Result().Cookies()[0].Value != cookie.Value {
		t.Fatal("player identity changed")
	}
	var identity map[string]string
	if err := json.Unmarshal(first.Body.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}
	if identity["ip"] != "203.0.113.1" {
		t.Fatal(identity)
	}
}

func TestConnectionIdentityWebSocketCookie(t *testing.T) {
	r := gin.New()
	r.Use(connectionIdentity)
	r.GET("/connect/:nickname", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, c.Writer.Header().Clone())
		if err == nil {
			conn.Close()
		}
	})
	server := httptest.NewServer(r)
	defer server.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/connect/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if len(response.Cookies()) != 1 || len(response.Cookies()[0].Value) != 32 {
		t.Fatal("missing browser identity cookie")
	}
}
