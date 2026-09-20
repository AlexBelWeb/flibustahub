-- Personal "want to read" flag, listing indexes, and partial indexes
-- for the unsynced counter and EXPORTABLE dump (slice 7).

ALTER TABLE works ADD COLUMN want_to_read INTEGER NOT NULL DEFAULT 0;
ALTER TABLE works ADD COLUMN want_to_read_updated_at TEXT;

-- Home carousel "My ratings" and catalog filter ?want=1.
CREATE INDEX idx_works_rated_at ON works(rating_updated_at DESC, id DESC) WHERE rating IS NOT NULL;
CREATE INDEX idx_works_want_at ON works(want_to_read_updated_at DESC, id DESC) WHERE want_to_read = 1;

-- EXPORTABLE: any personal timestamp set, including cleared values.
CREATE INDEX idx_works_rating_ts ON works(id) WHERE rating_updated_at IS NOT NULL;
CREATE INDEX idx_works_comment_ts ON works(id) WHERE comment_updated_at IS NOT NULL;
CREATE INDEX idx_works_want_ts ON works(id) WHERE want_to_read_updated_at IS NOT NULL;

-- Unsynced counter: timestamp newer than exported_at (or never exported).
CREATE INDEX idx_works_unsynced_rating ON works(id)
  WHERE rating_updated_at IS NOT NULL
    AND (exported_at IS NULL OR rating_updated_at > exported_at);
CREATE INDEX idx_works_unsynced_comment ON works(id)
  WHERE comment_updated_at IS NOT NULL
    AND (exported_at IS NULL OR comment_updated_at > exported_at);
CREATE INDEX idx_works_unsynced_want ON works(id)
  WHERE want_to_read_updated_at IS NOT NULL
    AND (exported_at IS NULL OR want_to_read_updated_at > exported_at);

-- Quick Tags: top series by work_count.
CREATE INDEX idx_series_work_count ON series(work_count DESC, id);

INSERT INTO app_meta(key, value)
VALUES
  ('authors_total', (SELECT count(*) FROM authors)),
  ('series_total', (SELECT count(*) FROM series))
ON CONFLICT(key) DO UPDATE SET value = excluded.value;
