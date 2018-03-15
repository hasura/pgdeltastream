package db

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gorilla/websocket"

	"github.com/hasura/pgdeltastream/types"
	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
)

var statusHeartbeatIntervalSeconds = 10

// LRStream will start streaming changes from the given slotName over the websocket connection
func LRStream(session *types.Session) {
	log.Infof("Starting replication for slot '%s' from LSN %s", session.SlotName, pgx.FormatLSN(session.RestartLSN))
	err := session.ReplConn.StartReplication(session.SlotName, session.RestartLSN, -1, "\"include-lsn\" 'on'", "\"pretty-print\" 'off'")
	if err != nil {
		log.Error(err)
		return
	}

	// start sending periodic status heartbeats to postgres
	go sendPeriodicHeartbeats(session)

	for {
		if !session.ReplConn.IsAlive() {
			log.WithField("CauseOfDeath", session.ReplConn.CauseOfDeath()).Error("Looks like the connection is dead")
		}
		log.Info("Waiting for LR message")

		ctx := session.Ctx
		message, err := session.ReplConn.WaitForReplicationMessage(ctx)
		if err != nil {
			// check whether the error is because of the context being cancelled
			if ctx.Err() != nil {
				// context cancelled, exit
				log.Warn("Websocket closed")
				return
			}

			log.WithError(err).Errorf("%s", reflect.TypeOf(err))
		}

		if message.WalMessage != nil {
			if message == nil {
				log.Error("Message nil")
				continue
			}
			walData := message.WalMessage.WalData
			log.Infof("Received replication message: %s", string(walData))

			// send message over ws
			// TODO: check if ws is open
			session.WSConn.WriteMessage(websocket.TextMessage, walData)
		}

		if message.ServerHeartbeat != nil {
			log.Info("Received server heartbeat")
			// set the flushed LSN (and other LSN values) in the standby status and send to PG
			log.Info(message.ServerHeartbeat)
			// send Standby Status if the server is requesting for a reply
			if message.ServerHeartbeat.ReplyRequested == 1 {
				log.Info("Status requested")
				err = sendStandbyStatus(session)
				if err != nil {
					log.WithError(err).Error("Unable to send standby status")
				}
			}
		}
	}
}

// LRListenAck listens on the websocket for ack messages
// The commited LSN is extracted and is updated to the server
func LRListenAck(session *types.Session, wsErr chan<- error) {
	jsonMsg := make(map[string]string)
	for {
		log.Info("Listening for WS message")
		//_, msg, err := session.WSConn.ReadMessage()
		err := session.WSConn.ReadJSON(&jsonMsg)
		if err != nil {
			log.WithError(err).Error("Error reading from websocket")
			wsErr <- err // send the error to the channel to terminate connection
			return
		}
		log.Info("Received WS message: ", jsonMsg)
		lsn := jsonMsg["lsn"]
		lrAckLSN(session, lsn)
	}
}

// sendStandbyStatus sends a StandbyStatus object with the current RestartLSN value to the server
func sendStandbyStatus(session *types.Session) error {
	standbyStatus, err := pgx.NewStandbyStatus(session.RestartLSN)
	if err != nil {
		return fmt.Errorf("unable to create StandbyStatus object: %s", err)
	}
	log.Info(standbyStatus)
	standbyStatus.ReplyRequested = 0
	log.Info("Sending Standby Status with LSN ", pgx.FormatLSN(session.RestartLSN))
	err = session.ReplConn.SendStandbyStatus(standbyStatus)
	if err != nil {
		return fmt.Errorf("unable to send StandbyStatus object: %s", err)
	}

	return nil
}

// send periodic keep alive hearbeats to the server so that the connection isn't dropped
func sendPeriodicHeartbeats(session *types.Session) {
	for {
		select {
		case <-session.Ctx.Done():
			// context closed; stop sending heartbeats
			return
		case <-time.Tick(time.Duration(statusHeartbeatIntervalSeconds) * time.Second):
			{
				// send hearbeat message at every statusHeartbeatIntervalSeconds interval
				log.Info("Sending periodic status heartbeat")
				err := sendStandbyStatus(session)
				if err != nil {
					log.WithError(err).Error("Failed to send status heartbeat")
				}
			}
		}
	}

}

// LRAckLSN will set the flushed LSN value and trigger a StandbyStatus update
func lrAckLSN(session *types.Session, restartLSNStr string) error {
	restartLSN, err := pgx.ParseLSN(restartLSNStr)
	if err != nil {
		return err
	}

	session.RestartLSN = restartLSN
	return sendStandbyStatus(session)
}
