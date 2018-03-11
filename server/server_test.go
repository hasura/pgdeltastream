package server

import (
	"fmt"
	"testing"

	"github.com/hasura/pgdeltastream/db"

	"github.com/jackc/pgx"

	"github.com/hasura/pgdeltastream/types"
)

func TestInit(t *testing.T) {
	session := &types.Session{}
	db.CreateConfig("postgres", "localhost")
	err := initDB(session)
	defer func() {
		// clean up
		if session.ReplConn != nil {
			session.ReplConn.DropReplicationSlot(session.SlotName)
			session.ReplConn.Close()
		}

		if session.PGConn != nil {
			session.PGConn.Close()
		}

	}()
	if err != nil {
		t.Fatal(err)
	}

	// check whether the slot has been created
	row := session.PGConn.QueryRow(
		fmt.Sprintf("select slot_name, plugin from pg_replication_slots where slot_name='%s'", session.SlotName))

	var slotname, plugin string
	err = row.Scan(&slotname, &plugin)
	if err != nil {
		if err == pgx.ErrNoRows {
			t.Fatal("failed to create replication slot: no rows match")
		}
		t.Fatal(err)
	}

	if slotname != session.SlotName {
		t.Errorf("Slotname doesn't match. Expected: %s, got: %s", session.SlotName, slotname)
	}

	if plugin != "wal2json" {
		t.Errorf("Plugin doesn't match. Expected: wal2json, got: %s", plugin)
	}

}

func TestSnapShotData(t *testing.T) {
	// first insert some test data into a test table
	conn, err := pgx.Connect(pgx.ConnConfig{
		Database: "postgres",
		Host:     "localhost",
	})
	defer conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	conn.Exec("CREATE TABLE delta_test(id serial primary key, name text)")
	defer conn.Exec("DROP TABLE delta_test")
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name1"))
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name2"))
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name3"))

	// init a replication slot
	session := &types.Session{}
	db.CreateConfig("postgres", "localhost")
	err = initDB(session)
	defer func() {
		// clean up
		if session.ReplConn != nil {
			session.ReplConn.DropReplicationSlot(session.SlotName)
			session.ReplConn.Close()
		}

		if session.PGConn != nil {
			session.PGConn.Close()
		}

	}()
	if err != nil {
		t.Fatalf("Could not init db: %s", err)
	}

	// insert more data _after_ creating the slot
	// these should not show up in the snapshot
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name4"))
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name5"))
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name6"))

	// get data as we would using the /snapshot/data endpoint
	data, err := snapshotData(session, "delta_test", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(data)

	if len(data) != 3 {
		t.Error("snapshot failed")
	}

	// match the data
	for _, d := range data {
		n := d["name"].(string)
		switch n {
		case "name1", "name2", "name3":
			continue
		case "name4", "name5", "name6":
			t.Errorf("invalid data in snapshot: %s", n)
		}
	}
}
