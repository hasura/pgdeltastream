package db

import (
	"context"
	"fmt"

	"github.com/hasura/pgdeltastream/types"
	log "github.com/sirupsen/logrus"

	"github.com/jackc/pgx"
)

func SnapshotInit() *types.Session {
	config := pgx.ConnConfig{
		Host:     "localhost",
		Database: "siddb",
	}

	replConn, err := pgx.ReplicationConnect(config)
	if err != nil {
		log.Error(err)
	}

	consistentPoint, snapshotName, err := replConn.CreateReplicationSlotEx("slot_ex", "wal2json") // TODO: slotName
	if err != nil {
		log.Error(err)
	}

	lsn, _ := pgx.ParseLSN(consistentPoint)
	session := types.Session{
		ReplConn:     replConn,
		RestartLSN:   lsn,
		SnapshotName: snapshotName,
	}
	log.Info(session.RestartLSN, " ", session.SnapshotName)
	return &session
}

func SnapshotData(session *types.Session) {
	//defer session.ReplConn.DropReplicationSlot("slot_ex")
	config := pgx.ConnConfig{
		Host:     "localhost",
		Database: "siddb",
	}

	conn, err := pgx.Connect(config)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := conn.BeginEx(context.TODO(), &pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})

	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec(fmt.Sprintf("SET TRANSACTION SNAPSHOT '%s'", session.SnapshotName))
	if err != nil {
		log.Fatal(err)
	}
	rows, err := tx.Query("SELECT * from test_table")
	if err != nil {
		log.Fatal(err)
	}
	//v, _ := rows.Values()
	//log.Info(v)
	//log.Info(rows.Values)
	for rows.Next() {
		var n int32
		var s string
		err = rows.Scan(&n, &s)
		if err != nil {
			log.Error(err)
		}
		log.Info(n, " ", s)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

}

/*
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

	session := types.Session{
		ReplConn: replConn,
	}

	//session.LRStream("test")
}
*/
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
