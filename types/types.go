package types

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx"
)

// Session stores the context, active db and ws connections, and replication slot state
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

// SnapshotDataJSON is the struct that binds with an incoming request for snapshot data
type SnapshotDataJSON struct {
	// SlotName is the name of the replication slot for which the snapshot data needs to be fetched
	// (not used as of now, will be useful in multi client setup)
	SlotName string `json:"slotName" binding:"omitempty"`

	Table  string `json:"table" binding:"required"`
	Offset *uint  `json:"offset" binding:"exists"`
	Limit  *uint  `json:"limit" binding:"exists"`
}

type Wal2JSONEvent struct {
	NextLSN string `json:"nextlsn"`
	Change  []map[string]interface{}
}

/*
type Wal2JSONChange struct{
	Kind string `json:"kind"`
	Schema string `json:"schema"`

}
*/
