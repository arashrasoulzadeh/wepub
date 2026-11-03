package store

import (
	"database/sql"

	"github.com/arashrasoulzadeh/wepub/server/internal/sync"
)

func AppendOps(db *sql.DB, ops []sync.Op) ([]int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	var seqs []int64
	for _, op := range ops {
		_, err := tx.Exec(`INSERT OR IGNORE INTO ops(id,device_id,hlc,entity,entity_id,field,value,created_at) VALUES (?,?,?,?,?,?,?,datetime('now'))`,
			op.ID, op.DeviceID, op.HLC, op.Entity, op.EntityID, op.Field, op.Value)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		var seq int64
		tx.QueryRow("SELECT seq FROM ops WHERE id=?", op.ID).Scan(&seq)
		seqs = append(seqs, seq)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return seqs, nil
}

func PullOps(db *sql.DB, cursor int64, limit int) ([]sync.Op, int64, bool, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := db.Query("SELECT seq,id,device_id,hlc,entity,entity_id,field,value FROM ops WHERE seq > ? ORDER BY seq ASC LIMIT ?", cursor, limit+1)
	if err != nil {
		return nil, cursor, false, err
	}
	defer rows.Close()
	var ops []sync.Op
	var lastSeq int64 = cursor
	count := 0
	hasMore := false
	for rows.Next() {
		var seq int64
		var op sync.Op
		var val []byte
		if err := rows.Scan(&seq, &op.ID, &op.DeviceID, &op.HLC, &op.Entity, &op.EntityID, &op.Field, &val); err != nil {
			return nil, cursor, false, err
		}
		op.Value = val
		if count < limit {
			ops = append(ops, op)
			lastSeq = seq
			count++
		} else {
			hasMore = true
		}
	}
	if len(ops) == 0 {
		lastSeq = cursor
	}
	return ops, lastSeq, hasMore, nil
}
