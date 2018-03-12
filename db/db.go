package db

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/hasura/pgdeltastream/types"
	log "github.com/sirupsen/logrus"

	"github.com/jackc/pgx"
)

var dbConfig = pgx.ConnConfig{}

// Initialize the database configuration
func CreateConfig(dbName, pgUser, pgHost string, pgPort int) {
	dbConfig.Database = dbName
	dbConfig.Host = pgHost
	dbConfig.Port = uint16(pgPort)
	dbConfig.User = pgUser
}

// Init function
// - creates a db connection
// - creates a replication connection
// - delete existing replication slots
// - creates a new replication slot
// - gets the consistent point LSN and snapshot name
// - populates the Session object
func Init(session *types.Session) error {

	// create a regular pg connection for use by transactions
	log.Info("Creating regular connection to db")
	pgConn, err := pgx.Connect(dbConfig)
	if err != nil {
		return err
	}

	session.PGConn = pgConn

	log.Info("Creating replication connection to ", dbConfig.Database)
	replConn, err := pgx.ReplicationConnect(dbConfig)
	if err != nil {
		return err
	}

	session.ReplConn = replConn

	// delete all existing slots
	err = deleteAllSlots(session)
	if err != nil {
		log.WithError(err).Error("could not delete replication slots")
	}

	// create new slots
	slotName := generateSlotName()
	session.SlotName = slotName

	log.Info("Creating replication slot ", slotName)
	consistentPoint, snapshotName, err := session.ReplConn.CreateReplicationSlotEx(slotName, "wal2json")
	if err != nil {
		return err
	}

	log.Infof("Created replication slot \"%s\" with consistent point LSN = %s, snapshot name = %s",
		slotName, consistentPoint, snapshotName)

	lsn, _ := pgx.ParseLSN(consistentPoint)

	session.RestartLSN = lsn
	session.SnapshotName = snapshotName

	return nil
}

func CheckAndCreateReplConn(session *types.Session) error {
	if session.ReplConn != nil {
		if session.ReplConn.IsAlive() {
			// reuse the existing connection (or close it nonetheless?)
			return nil
		}
	}

	replConn, err := pgx.ReplicationConnect(dbConfig)
	if err != nil {
		return err
	}
	session.ReplConn = replConn

	return nil
}

// generates a random slot name which can be remembered
func generateSlotName() string {
	// list of random words
	strs := []string{
		"gigantic",
		"scold",
		"greasy",
		"shaggy",
		"wasteful",
		"few",
		"face",
		"pet",
		"ablaze",
		"mundane",
	}

	rand.Seed(time.Now().Unix())

	// generate name such as gigantic20
	name := fmt.Sprintf("delta_%s%d", strs[rand.Intn(len(strs))], rand.Intn(100))

	return name
}

// delete all old slots that were created by us
func deleteAllSlots(session *types.Session) error {
	rows, err := session.PGConn.Query("SELECT slot_name FROM pg_replication_slots")
	if err != nil {
		return err
	}
	for rows.Next() {
		var slotName string
		rows.Scan(&slotName)

		// only delete slots created by this program
		if !strings.Contains(slotName, "delta_") {
			continue
		}

		log.Infof("Deleting replication slot %s", slotName)
		err = session.ReplConn.DropReplicationSlot(slotName)
		//_,err = session.PGConn.Exec(fmt.Sprintf("SELECT pg_drop_replication_slot(\"%s\")", slotName))
		if err != nil {
			log.WithError(err).Error("could not delete slot ", slotName)
		}
	}
	return nil
}
