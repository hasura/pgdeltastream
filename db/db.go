package db

import (
	"context"

	"github.com/hasura/pgdeltastream/types"
	log "github.com/sirupsen/logrus"

	"github.com/jackc/pgx"
)

var config = pgx.ConnConfig{
	Host:     "localhost",
	Database: "siddb",
}

// Init function
// - creates a replication connection
// - creates a replication slot
// - gets the consistent point LSN and snapshot name
// - creates a db connection
// - finally returns a Session object containing the above
func Init(session *types.Session) error {
	log.Info("Creating replication connection to ", config.Database)
	replConn, err := pgx.ReplicationConnect(config)
	if err != nil {
		return err
	}

	session.ReplConn = replConn

	slotName := generateSlotName()
	session.SlotName = slotName

	log.Info("Creating replication slot ", slotName)
	consistentPoint, snapshotName, err := replConn.CreateReplicationSlotEx(slotName, "wal2json")
	if err != nil {
		return err
	}

	log.Infof("Created replication slot \"%s\" with consistent point LSN = %s, snapshot name = %s",
		slotName, consistentPoint, snapshotName)

	lsn, _ := pgx.ParseLSN(consistentPoint)

	session.RestartLSN = lsn
	session.SnapshotName = snapshotName

	// create a regular pg connection for use by transactions
	log.Info("Creating regular connection to db")
	pgConn, err := pgx.Connect(config)
	if err != nil {
		return err
	}

	session.PGConn = pgConn

	return nil
}

func CheckAndCreateReplConn(session *types.Session) error {
	if session.ReplConn != nil {
		if session.ReplConn.IsAlive() {
			// reuse the existing connection (or close it nonetheless?)
			return nil
		}
	}

	replConn, err := pgx.ReplicationConnect(config)
	if err != nil {
		return err
	}
	session.ReplConn = replConn

	return nil
}

func generateSlotName() string {
	return "slot_ex"
}

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
