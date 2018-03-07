package main

import "github.com/hasura/pgdeltastream/db"

func main() {
	// args: dbname, server hostport
	//server.StartServer()
	db.DBConnect()
}
