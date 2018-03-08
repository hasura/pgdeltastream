package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hasura/pgdeltastream/db"
	"github.com/hasura/pgdeltastream/types"
)

func StartServer() {
	var session types.Session
	r := gin.Default()
	v1 := r.Group("/v1")
	{
		snapshotRoute := v1.Group("/snapshot")
		{
			snapshotRoute.GET("/init", func(c *gin.Context) {
				// create replication slot ex and get snapshot name, consistent point
				// return slotname (so that any kind of failure can be restarted from)
				c.String(200, "init") // TODO
			})

			snapshotRoute.POST("/data", func(c *gin.Context) {
				// get data with table, offset, limits
				c.String(200, "get data") // TODO
			})

			snapshotRoute.GET("/end", func(c *gin.Context) {
				// end snapshot
				c.String(200, "end") // TODO
			})
		}

		lrRoute := v1.Group("/lr")
		{
			lrRoute.GET("/stream", func(c *gin.Context) { // /stream?slotName=my_slot
				// start streaming from slot name
				slotName := "test" // TODO extract from url params

				session.WSConn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
				db.LRStream(session, slotName) // err?

			})
		}
	}

	r.Run("localhost:12312")
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


/*
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
*/