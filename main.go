package main

import (
	"os"
	"strconv"

	"github.com/hasura/pgdeltastream/db"
	"github.com/hasura/pgdeltastream/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	// args: dbname, pguser, pgpass, pghost, pgport, server host, server port
	if len(os.Args) != 8 {
		log.Fatal("Run with args: dbName pgUser pgPass pgHost pgPort serverHost serverPort")
	}
	dbName := os.Args[1]
	pgUser := os.Args[2]
	pgPass := os.Args[3]
	pgHost := os.Args[4]
	pgPort, _ := strconv.Atoi(os.Args[5])
	serverHost := os.Args[6]
	serverPort, _ := strconv.Atoi(os.Args[7])

	db.CreateConfig(dbName, pgUser, pgPass, pgHost, pgPort)

	log.Infof("Starting server for database %s; serving at %s:%d", dbName, serverHost, serverPort)
	server.StartServer(serverHost, serverPort)
}
