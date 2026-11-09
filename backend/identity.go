package main

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func connectionIdentity(c *gin.Context) {
	if !strings.Contains(c.FullPath(), "/connect/") && !strings.Contains(c.FullPath(), "/reconnect/") {
		return
	}
	identity, _ := c.Cookie("fiddlesticks_player")
	if decoded, err := hex.DecodeString(identity); err != nil || len(decoded) != 16 {
		identity = hex.EncodeToString(randBytes())
	}
	c.SetCookie("fiddlesticks_player", identity, 365*24*3600, "/", "", !DEV, true)
	ip := c.ClientIP()
	forwarded := strings.Split(c.GetHeader("X-Forwarded-For"), ",")
	if !DEV && len(forwarded) >= 2 {
		ip = strings.TrimSpace(forwarded[len(forwarded)-2])
	}
	c.Set("identity", map[string]string{"player_id": identity, "ip": ip, "user_agent": c.Request.UserAgent(), "connected_at": time.Now().UTC().Format(time.RFC3339Nano)})
}

func randBytes() []byte {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return value
}
