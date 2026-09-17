package repositories

import (
	"context"
	"database/sql"
)

// InsertRunningBatch records that an import has started on the write connection.
func InsertRunningBatch(ctx context.Context, conn *sql.Conn, startedAt, status, path string) (int64, error) {
	var id int64
	err := conn.QueryRowContext(ctx, `INSERT INTO import_batches (started_at, status, inpx_path) VALUES (?, ?, ?) RETURNING id`,
		startedAt, status, path).Scan(&id)
	return id, err
}

// SetBatchVersion stores the dump version on a running batch.
func SetBatchVersion(ctx context.Context, conn *sql.Conn, id int64, version string) error {
	_, err := conn.ExecContext(ctx, `UPDATE import_batches SET inpx_version = ? WHERE id = ?`, version, id)
	return err
}

// BatchFinish is the final import_batches row written after COMMIT or ROLLBACK.
type BatchFinish struct {
	ID                  int64
	Status              string
	FinishedAt          string
	INPXVersion         string
	RecordsSeen         int
	WorksAdded          int
	EditionsAdded       int
	EditionsUpdated     int
	EditionsDeactivated int
	LibIDCollisions     int
	NotesJSON           string
}

// FinishBatch writes the terminal import_batches row on the write connection.
func FinishBatch(ctx context.Context, conn *sql.Conn, b BatchFinish) error {
	_, err := conn.ExecContext(ctx, `UPDATE import_batches SET
		finished_at = ?, status = ?, inpx_version = ?, records_seen = ?, works_added = ?,
		editions_added = ?, editions_updated = ?, editions_deactivated = ?, libid_collisions = ?, notes = ?
		WHERE id = ?`,
		b.FinishedAt, b.Status, b.INPXVersion, b.RecordsSeen, b.WorksAdded,
		b.EditionsAdded, b.EditionsUpdated, b.EditionsDeactivated, b.LibIDCollisions, b.NotesJSON, b.ID)
	return err
}
