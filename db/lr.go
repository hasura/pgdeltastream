package db

import (
	"context"
	"reflect"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
)

func (session *Session) LRStream(slotName string) {

	err := session.ReplConn.StartReplication(slotName, session.RestartLSN, -1, "\"include-lsn\" 'on'", "\"pretty-print\" 'off'")
	if err != nil {
		log.Error(err)
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

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
				err = session.sendStandbyStatus()
				if err != nil {
					log.WithError(err).Error("Unable to send standby status")
				}
			}
		}
	}
}

func (session *Session) sendStandbyStatus() error {
	standbyStatus, err := pgx.NewStandbyStatus(session.RestartLSN)
	if err != nil {
		log.WithError(err).Error("Unable to create StandbyStatus object")
		return err
	}
	log.Info(standbyStatus)
	standbyStatus.ReplyRequested = 0
	err = session.ReplConn.SendStandbyStatus(standbyStatus)
	if err != nil {
		return err
	}
	return nil
}

// set the flushed LSN value
func (session *Session) LRAckLSN(restartLSNStr string) error {
	restartLSN, err := pgx.ParseLSN(restartLSNStr)
	if err != nil {
		return err
	}

	session.RestartLSN = restartLSN
	return session.sendStandbyStatus()
}
