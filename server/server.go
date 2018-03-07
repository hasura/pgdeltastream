package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hasura/pgdeltastream/db"
	log "github.com/sirupsen/logrus"
)

func StartServer() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(200, "We got Gin")
	})
	r.GET("/ws", func(c *gin.Context) {
		wshandler(c.Writer, c.Request)
	})
	log.Info("dfs")
	r.Run("localhost:12312")
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		db.LRAckLSN("ee")
	}
}
