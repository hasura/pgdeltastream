package main

import "github.com/hasura/pgdeltastream/server"

func main() {
	// args: dbname, server hostport
	server.StartServer()
	//db.DBConnect()
	//session := db.SnapshotInit()
	//db.SnapshotData(session, "test_table", 0, 100)
	//db.SnapshotData(session)
	for {
	}
}
