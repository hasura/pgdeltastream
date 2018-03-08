package server

import (
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
					c.AbortWithError(400, err)
					return
				}

				log.Infof("Snapshot data requested for table: %s, offset: %d, limit: %d", postData.Table, postData.Offset, postData.Limit)

				data, err := db.SnapshotData(session, postData.Table, postData.Offset, postData.Limit)
				if err != nil {
					log.WithError(err).Error("Unable to get snapshot data")
					c.AbortWithError(500, err)
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
				slotName := "test" // TODO extract from url params
				wsConn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					log.Error(err)
				}
				session.WSConn = wsConn
				go db.LRListenAck(session)     // concurrently listen for ack messages
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
