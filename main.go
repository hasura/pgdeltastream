package main

import (
	"os"
	"strconv"

	"github.com/hasura/pgdeltastream/db"
	"github.com/hasura/pgdeltastream/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	// args: dbname, pguser, pghost, pgport, server host server port
	if len(os.Args) != 7 {
		log.Fatal("Run with args: dbName pgUser pgHost pgPort serverHost serverPort")
	}
	dbName := os.Args[1]
	pgUser := os.Args[2]
	pgHost := os.Args[3]
	pgPort, _ := strconv.Atoi(os.Args[4])
	serverHost := os.Args[5]
	serverPort, _ := strconv.Atoi(os.Args[6])

	db.CreateConfig(dbName, pgUser, pgHost, pgPort)

	log.Infof("Starting server for database %s; serving at %s:%d", dbName, serverHost, serverPort)
	server.StartServer(serverHost, serverPort)
}
