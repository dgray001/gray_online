package game

import "github.com/gin-gonic/gin"

type Viewer struct {
	client_id uint64
	nickname  string
	connected bool
	Updates   chan *UpdateMessage
}

func CreateViewer(client_id uint64, nickname string) *Viewer {
	return &Viewer{
		client_id: client_id,
		nickname:  nickname,
		connected: false,
		Updates:   make(chan *UpdateMessage),
	}
}

func (v *Viewer) ToFrontend() gin.H {
	return gin.H{
		"client_id": v.client_id,
		"nickname":  v.nickname,
		"connected": v.connected,
	}
}
