package types

import (
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx"
)

type Session struct {
	RestartLSN   uint64
	ReplConn     *pgx.ReplicationConn
	WSConn       *websocket.Conn
	SnapshotName string
}
