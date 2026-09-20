// Package visibility holds catalog listing rules shared by import warmup and search.
package visibility

// VisibleEditionSQL is the physical-presence predicate on editions aliased as e.
const VisibleEditionSQL = `e.is_active = 1 AND e.is_deleted = 0`

// ListableWorkSQL is the catalog-listing predicate on works aliased as w:
// a work stays listed if it has a visible edition or personal value.
const ListableWorkSQL = `(
  EXISTS (
    SELECT 1 FROM editions e
     WHERE e.work_id = w.id AND e.is_active = 1 AND e.is_deleted = 0
  )
  OR w.rating IS NOT NULL
  OR (w.comment IS NOT NULL AND trim(w.comment) != '')
  OR w.want_to_read = 1
)`

// ExportableWorkSQL is the personal-history predicate on works aliased as w.
// A work is dumped if any personal timestamp is set, including cleared values.
const ExportableWorkSQL = `(
  w.rating_updated_at IS NOT NULL
  OR w.comment_updated_at IS NOT NULL
  OR w.want_to_read_updated_at IS NOT NULL
)`

// ListableTempSQL builds a keyed temp table of listable work ids for set-based aggregates.
var ListableTempSQL = []string{
	`DROP TABLE IF EXISTS temp.listable`,
	`CREATE TEMP TABLE listable (work_id INTEGER PRIMARY KEY)`,
	`INSERT OR IGNORE INTO listable(work_id)
	 SELECT DISTINCT work_id FROM editions WHERE is_active = 1 AND is_deleted = 0`,
	`INSERT OR IGNORE INTO listable(work_id)
	 SELECT id FROM works
	  WHERE rating IS NOT NULL
	     OR (comment IS NOT NULL AND trim(comment) <> '')
	     OR want_to_read = 1`,
}
