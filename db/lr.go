package db

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gorilla/websocket"

	"github.com/hasura/pgdeltastream/types"
	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
)

// LRStream will start streaming changes from the given slotName over the websocket connection
func LRStream(session *types.Session, slotName string) {

	err := session.ReplConn.StartReplication(slotName, session.RestartLSN, -1, "\"include-lsn\" 'on'", "\"pretty-print\" 'off'")
	if err != nil {
		log.Error(err)
	}
	//ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	//defer cancelFn()
	ctx := context.TODO()
	for {
		log.Info("Waiting for message")

		message, err := session.ReplConn.WaitForReplicationMessage(ctx)
		if err != nil {
			log.WithError(err).Errorf("%s", reflect.TypeOf(err))
		}

		if message.WalMessage != nil {

			//log.Info(pgx.FormatLSN(message.WalMessage.WalStart))
			log.Info(string(message.WalMessage.WalData))
			walData := message.WalMessage.WalData
			session.WSConn.WriteMessage(websocket.TextMessage, walData)
		}

		if message.ServerHeartbeat != nil {
			log.Info("Heartbeat requested")
			// set the flushed LSN (and other LSN values) in the standby status and send to PG
			log.Info(message.ServerHeartbeat)
			// send Standby Status if the server is requesting for a reply
			if message.ServerHeartbeat.ReplyRequested == 1 {
				err = sendStandbyStatus(session)
				if err != nil {
					log.WithError(err).Error("Unable to send standby status")
				}
			}
		}

		_, msg, err := session.WSConn.ReadMessage()
		if err != nil {
			log.WithError(err).Error("Error reading from websocket")
			//break
		}
		processWSMessage(session, msg) // TODO
	}
}

func processWSMessage(session *types.Session, msg []byte) {

	//LRAckLSN(session, restartLSNStr)
}

// sendStandbyStatus sends a StandbyStatus object with the current RestartLSN value to the server
func sendStandbyStatus(session *types.Session) error {
	standbyStatus, err := pgx.NewStandbyStatus(session.RestartLSN)
	if err != nil {
		return fmt.Errorf("Unable to create StandbyStatus object: %s", err)
	}
	log.Info(standbyStatus)
	standbyStatus.ReplyRequested = 0
	err = session.ReplConn.SendStandbyStatus(standbyStatus)
	if err != nil {
		return fmt.Errorf("Unable to send StandbyStatus object: %s", err)
	}

	return nil
}

// LRAckLSN will set the flushed LSN value and trigger a StandbyStatus update
func LRAckLSN(session *types.Session, restartLSNStr string) error {
	restartLSN, err := pgx.ParseLSN(restartLSNStr)
	if err != nil {
		return err
	}

	session.RestartLSN = restartLSN
	return sendStandbyStatus(session)
}
