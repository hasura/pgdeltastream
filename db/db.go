package db

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/jackc/pgx"
)

func DBConnect() {
	config := pgx.ConnConfig{
		Host:     "localhost",
		Database: "siddb",
	}

	replConn, err := pgx.ReplicationConnect(config)
	defer replConn.Close()
	if err != nil {
		log.Error(err)
	}

	session := Session{
		ReplConn: replConn,
	}

	session.LRStream("test")
}

// start the LR stream

func DBConnect1() {
	config := pgx.ConnConfig{
		Host:     "localhost",
		Database: "siddb",
	}

	replConn, err := pgx.ReplicationConnect(config)
	defer replConn.Close()
	if err != nil {
		log.Error(err)
	}
	err = replConn.CreateReplicationSlot("my_slot", "wal2json")
	if err != nil {
		log.Error(err)
	}
	restartLsn, _ := pgx.ParseLSN("0/15E9108")
	err = replConn.StartReplication("test", restartLsn, -1, "\"include-lsn\" 'on'", "\"pretty-print\" 'off'")
	if err != nil {
		log.Error(err)
	}

	for {
		log.Info("Waiting for message")
		message, err := replConn.WaitForReplicationMessage(context.TODO())
		if err != nil {
			log.Error(err)
		}

		if message.WalMessage != nil {
			log.Info(pgx.FormatLSN(message.WalMessage.WalStart))
			log.Info(string(message.WalMessage.WalData))
		}
	}

	//c, _ := replConn.Exec("select * from test_table")
	//fmt.Println(c)

}
