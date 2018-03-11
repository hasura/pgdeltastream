package types

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx"
)

// Session stores the active db and ws connections, and replication slot state
type Session struct {
	Ctx        context.Context
	CancelFunc context.CancelFunc

	ReplConn *pgx.ReplicationConn
	PGConn   *pgx.Conn

	WSConn *websocket.Conn

	SlotName     string
	SnapshotName string
	RestartLSN   uint64
}

type SnapshotDataJSON struct {
	Table  string `json:"table"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}
