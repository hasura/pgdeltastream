package db

import (
	"context"
	"fmt"

	"github.com/hasura/pgdeltastream/types"
	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
)

// SnapshotData queries the snapshot for data from the given table
// and returns the results as a JSON array
func SnapshotData(session *types.Session, requestParams *types.SnapshotDataJSON) ([]map[string]interface{}, error) {
	log.Info("Begin transaction")
	tx, err := session.PGConn.BeginEx(context.TODO(), &pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})

	if err != nil {
		return nil, err
	}

	log.Info("Setting transaction snapshot ", session.SnapshotName)
	_, err = tx.Exec(fmt.Sprintf("SET TRANSACTION SNAPSHOT '%s'", session.SnapshotName))
	if err != nil {
		return nil, err
	}

	var query string
	if requestParams.OrderBy == nil {
		// no ordering specified by user
		query = fmt.Sprintf("SELECT * FROM %s OFFSET %d LIMIT %d", requestParams.Table, *requestParams.Offset, *requestParams.Limit)
	} else {
		query = fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s OFFSET %d LIMIT %d", requestParams.Table, requestParams.OrderBy.Column, requestParams.OrderBy.Order, *requestParams.Offset, *requestParams.Limit)
	}
	log.Info("Executing query: ", query)
	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}
	// convert data each row into columnName, value map
	data := processRows(rows)
	log.Info("Number of results: ", len(data))

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// put each row in a map containing column names and values
func processRows(rows *pgx.Rows) []map[string]interface{} {
	fields := rows.FieldDescriptions()
	resultsList := make([]map[string]interface{}, 0)
	for rows.Next() {
		values, _ := rows.Values()
		rowJSON := make(map[string]interface{})
		for i := 0; i < len(fields); i++ {
			name := fields[i].Name
			value := values[i]
			rowJSON[name] = value
		}
		resultsList = append(resultsList, rowJSON)
	}

	return resultsList
}
