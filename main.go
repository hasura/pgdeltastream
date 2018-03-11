package main

import (
	"os"

	"github.com/hasura/pgdeltastream/db"
	"github.com/hasura/pgdeltastream/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	// args: dbname, dbhost, server hostport
	if len(os.Args) != 4 {
		log.Fatal("Run with args: dbName dbHost serverHostPort")
	}
	dbName := os.Args[1]
	dbHost := os.Args[2]
	serverHostPort := os.Args[3]

	db.CreateConfig(dbName, dbHost)

	log.Infof("Starting server for database %s; serving at %s", dbName, serverHostPort)
	server.StartServer(serverHostPort)
}
