package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hasura/pgdeltastream/db"
	"github.com/hasura/pgdeltastream/types"
	log "github.com/sirupsen/logrus"
)

func StartServer() {
	var session *types.Session
	r := gin.Default()
	v1 := r.Group("/v1")
	{
		v1.GET("/init", func(c *gin.Context) {
			// create replication slot ex and get snapshot name, consistent point
			// return slotname (so that any kind of failure can be restarted from)

			// TODO: close the session before initiating a new one
			session = db.Init()

			c.JSON(200, gin.H{
				"slotName": session.SlotName,
			})
		})

		snapshotRoute := v1.Group("/snapshot")
		{

			snapshotRoute.POST("/data", func(c *gin.Context) {
				// get data with table, offset, limits
				var postData types.SnapshotDataJSON
				err := c.ShouldBindJSON(&postData)
				if err != nil {
					log.WithError(err).Error("Invalid input JSON")
					c.AbortWithError(http.StatusBadRequest, err)
					return
				}

				log.Infof("Snapshot data requested for table: %s, offset: %d, limit: %d", postData.Table, postData.Offset, postData.Limit)

				data, err := db.SnapshotData(session, postData.Table, postData.Offset, postData.Limit)
				if err != nil {
					log.WithError(err).Error("Unable to get snapshot data")
					c.AbortWithError(http.StatusInternalServerError, err)
					return
				}
				c.JSON(200, data)
			})

			/*
				snapshotRoute.GET("/end", func(c *gin.Context) {
					// end snapshot
					c.String(200, "end") // TODO
				})
			*/
		}

		lrRoute := v1.Group("/lr")
		{
			lrRoute.GET("/stream", func(c *gin.Context) { // /stream?slotName=my_slot
				// start streaming from slot name
				slotName := c.Query("slotName")
				if slotName == "" {
					e := fmt.Errorf("No slotName provided")
					log.Error(e)
					c.AbortWithError(http.StatusBadRequest, e)
					return
				}

				log.Info("LR Stream requested for slot ", slotName)
				session.SlotName = slotName

				// now upgrade the HTTP  connection to a websocket connection
				wsConn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					log.Error(err)
				}
				session.WSConn = wsConn

				ctx, cancelFunc := context.WithCancel(context.Background())
				wsErr := make(chan error, 1)
				go db.LRListenAck(session, wsErr) // concurrently listen for ack messages
				go db.LRStream(session, ctx)      // err?

				select {
				/*case <-c.Writer.CloseNotify(): // ws closed
				log.Warn("Websocket connection closed. Cancelling context.")
				cancelFunc()
				*/
				case <-wsErr: // ws closed
					log.Warn("Websocket connection closed. Cancelling context.")
					cancelFunc()
				}

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
