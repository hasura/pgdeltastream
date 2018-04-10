package main

import (
	"flag"

	"github.com/hasura/pgdeltastream/db"
	"github.com/hasura/pgdeltastream/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	// args: dbname, pguser, pgpass, pghost, pgport, server host, server port
	/*
		if len(os.Args) != 8 {
			log.Fatal("Run with args: dbName pgUser pgPass pgHost pgPort serverHost serverPort")
		}
	*/
	dbName := flag.String("db", "hasuradb", "Name of the database to connect to") //os.Args[1]
	pgUser := flag.String("user", "admin", "Postgres user name")                  //os.Args[2]
	pgPass := flag.String("password", "", "Postgres password")                    //os.Args[3]
	pgHost := flag.String("pgHost", "localhost", "Postgres server hostname")      //os.Args[4]
	pgPort := flag.Int("pgPort", 5432, "Postgres server port")                    //strconv.Atoi(os.Args[5])
	serverHost := flag.String("serverHost", "0.0.0.0", "Host to listen on")       //os.Args[6]
	serverPort := flag.Int("serverPort", 8080, "Port to listen on")               //strconv.Atoi(os.Args[7])
	flag.Parse()
	db.CreateConfig(*dbName, *pgUser, *pgPass, *pgHost, *pgPort)

	log.Infof("Starting server for database %s; serving at %s:%d", *dbName, *serverHost, *serverPort)
	server.StartServer(*serverHost, *serverPort)
}
