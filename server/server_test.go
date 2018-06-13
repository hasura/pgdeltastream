package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/hasura/pgdeltastream/db"

	"github.com/jackc/pgx"

	"github.com/hasura/pgdeltastream/types"
)

func TestInit(t *testing.T) {
	session := &types.Session{}
	db.CreateConfig("postgres", "postgres", "", "localhost", 5432)
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
	db.CreateConfig("postgres", "postgres", "", "localhost", 5432)
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
	offset := uint(0)
	limit := uint(10)
	requestJSON := types.SnapshotDataJSON{
		Table:  "delta_test",
		Offset: &offset,
		Limit:  &limit,
		OrderBy: &types.OrderBy{
			Column: "id",
			Order:  "ASC",
		},
	}
	data, err := snapshotData(session, &requestJSON)
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

func TestLRWebsoscket(t *testing.T) {
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

	// start the server
	go StartServer("localhost", 8181)

	// init a replication slot
	r, err := http.Get("http://localhost:8181/v1/init")
	if err != nil {
		t.Fatal("could not init: ", err)
	}
	if r.StatusCode != 200 {
		t.Fatal("could not init: ", r.StatusCode)
	}
	defer r.Body.Close()

	resp := make(map[string]string)
	err = json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		t.Error("could not decode json")
	}

	slotName := resp["slotName"]

	// start the LR stream
	ws, _, err := websocket.DefaultDialer.
		Dial(fmt.Sprintf("ws://localhost:8181/v1/lr/stream?slotName=%s", slotName), nil)
	if err != nil {
		t.Fatal("ws dial:", err)
	}
	defer ws.Close()

	// let's insert some data that should show up on the stream
	conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name4"))
	w2jEvent := types.Wal2JSONEvent{}
	err = ws.ReadJSON(&w2jEvent)
	if err != nil {
		t.Fatal("couldn't parse json")
	}
	//fmt.Println(w2jEvent)
	c := w2jEvent.Change[0]
	if c["kind"] != "insert" {
		log.Fatalf("change type doesn't match. expected: %s, got: %s", "insert", c["kind"])
	}

	colVals := c["columnvalues"].([]interface{}) // it can be string, int, bool, etc
	if colVals[1].(string) != "name4" {
		t.Fatal("ws data doesn't match")
	}

	// let's acknowledge this event and test
	nextLSN := w2jEvent.NextLSN
	err = ws.WriteJSON(map[string]string{
		"lsn": nextLSN,
	})
	if err != nil {
		t.Fatal("could not write to ws: ", err)
	}

	flushedLSN := getConfirmedFlushLsnFor(t, conn, slotName)
	//fmt.Printf("###########%s %s\n", flushedLSN, nextLSN)
	if flushedLSN != nextLSN {
		t.Fatal("LSN not committed")
	}

	// run update query
	conn.Exec(fmt.Sprintf("update delta_test set name = '%s' where id = %d", "name_new", 1))
	w2jEvent2 := types.Wal2JSONEvent{}
	err = ws.ReadJSON(&w2jEvent2)
	if err != nil {
		t.Fatal("couldn't parse json")
	}
	//fmt.Println(w2jEvent)
	c = w2jEvent2.Change[0]

	if c["kind"] != "update" {
		log.Fatalf("change type doesn't match. expected: %s, got: %s", "update", c["kind"])
	}

	colVals = c["columnvalues"].([]interface{}) // it can be string, int, bool, etc
	// match id, name
	if colVals[0].(float64) != 1 && colVals[1].(string) != "name_new" {
		t.Fatal("ws data doesn't match")
	}

	// let's acknowledge this event and test
	nextLSN = w2jEvent2.NextLSN
	err = ws.WriteJSON(map[string]string{
		"lsn": nextLSN,
	})
	if err != nil {
		t.Fatal("could not write to ws: ", err)
	}

	flushedLSN = getConfirmedFlushLsnFor(t, conn, slotName)
	//fmt.Printf("###########%s %s\n", flushedLSN, nextLSN)
	if flushedLSN != nextLSN {
		t.Fatal("LSN not committed")
	}

	//conn.Exec(fmt.Sprintf("insert into delta_test(name) values('%s')", "name6"))

	// we're done. close ws connection.
	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Error("could not close ws: ", err)
	}
}

func getConfirmedFlushLsnFor(t *testing.T, conn *pgx.Conn, slot string) string {
	// Fetch the restart LSN of the slot, to establish a starting point
	rows, err := conn.Query(fmt.Sprintf("select confirmed_flush_lsn from pg_replication_slots where slot_name='%s'", slot))
	if err != nil {
		t.Fatalf("conn.Query failed: %v", err)
	}
	defer rows.Close()

	var restartLsn string
	for rows.Next() {
		rows.Scan(&restartLsn)
	}
	return restartLsn
}
