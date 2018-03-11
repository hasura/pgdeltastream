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

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func StartServer(hostPort string) {
	var err error
	session := &types.Session{}
	r := gin.Default()
	v1 := r.Group("/v1")
	{
		v1.GET("/init", func(c *gin.Context) {
			err = initDB(session)
			if err != nil {
				e := fmt.Sprintf("unable to init session")
				log.WithError(err).Error(e)
				c.AbortWithStatusJSON(http.StatusInternalServerError, e)
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"slotName": session.SlotName,
			})
		})

		snapshotRoute := v1.Group("/snapshot")
		{

			snapshotRoute.POST("/data", func(c *gin.Context) {
				if session.SnapshotName == "" {
					e := fmt.Sprintf("snapshot not available: call /init to initialize a new slot and snapshot")
					log.Error(e)
					c.AbortWithStatusJSON(http.StatusServiceUnavailable, e)
					return
				}

				// get data with table, offset, limits
				var postData types.SnapshotDataJSON
				err = c.ShouldBindJSON(&postData)
				if err != nil {
					e := fmt.Sprintf("invalid input JSON")
					log.WithError(err).Error(e)
					c.AbortWithStatusJSON(http.StatusBadRequest, e)
					return
				}

				log.Infof("Snapshot data requested for table: %s, offset: %d, limit: %d", postData.Table, postData.Offset, postData.Limit)

				data, err := snapshotData(session, postData.Table, postData.Offset, postData.Limit)
				if err != nil {
					e := fmt.Sprintf("unable to get snapshot data")
					log.WithError(err).Error(e)
					c.AbortWithStatusJSON(http.StatusInternalServerError, e)
					return
				}

				c.JSON(http.StatusOK, data)
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
				slotName := c.Query("slotName")
				if slotName == "" {
					e := fmt.Sprintf("no slotName provided")
					log.WithError(err).Error(e)
					c.AbortWithStatusJSON(http.StatusBadRequest, e)
					return
				}

				log.Info("LR Stream requested for slot ", slotName)
				session.SlotName = slotName

				// now upgrade the HTTP  connection to a websocket connection
				wsConn, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					e := fmt.Sprintf("could not upgrade to websocket connection")
					log.WithError(err).Error(e)
					c.AbortWithStatusJSON(http.StatusInternalServerError, e)
					return
				}
				session.WSConn = wsConn

				// begin streaming
				err = lrStream(session)
				if err != nil {
					e := fmt.Sprintf("could not create stream")
					log.WithError(err).Error(e)
					c.AbortWithStatusJSON(http.StatusInternalServerError, e)
					return
				}
			})
		}
	}

	r.Run(hostPort)
}

func initDB(session *types.Session) error {
	// create replication slot ex and get snapshot name, consistent point
	// return slotname (so that any kind of failure can be restarted from)
	var err error
	// initilize the connections for the session
	resetSession(session)
	err = db.Init(session)
	if err != nil {
		return err
	}

	return nil
}

func snapshotData(session *types.Session, tableName string, offset, limit int) ([]map[string]interface{}, error) {
	return db.SnapshotData(session, tableName, offset, limit)
}

func lrStream(session *types.Session) error {
	// reset the connections
	err := resetSession(session)
	if err != nil {
		log.WithError(err).Error("Could not create replication connection")
		return fmt.Errorf("Could not create replication connection")
	}

	//ctx, cancelFunc := context.WithCancel(context.Background())
	wsErr := make(chan error, 1)
	go db.LRListenAck(session, wsErr) // concurrently listen for ack messages
	go db.LRStream(session)           // err?

	select {
	/*case <-c.Writer.CloseNotify(): // ws closed
	  log.Warn("Websocket connection closed. Cancelling context.")
	  cancelFunc()
	*/
	case <-wsErr: // ws closed
		log.Warn("Websocket connection closed. Cancelling context.")
		// cancel session context
		session.CancelFunc()
		// close connections
		err = session.WSConn.Close()
		if err != nil {
			log.WithError(err).Error("Could not close websocket connection")
		}

		err = session.ReplConn.Close()
		if err != nil {
			log.WithError(err).Error("Could not close replication connection")
		}

	}
	return nil
}

// Cancel the currently running session
// Recreate replication connection
func resetSession(session *types.Session) error {
	var err error
	// cancel the currently running session
	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	// close websocket connection
	if session.WSConn != nil {
		//err = session.WSConn.Close()
		if err != nil {
			return err
		}
	}

	// create new context
	ctx, cancelFunc := context.WithCancel(context.Background())
	session.Ctx = ctx
	session.CancelFunc = cancelFunc

	// create the replication connection
	err = db.CheckAndCreateReplConn(session)
	if err != nil {
		return err
	}

	return nil

}
