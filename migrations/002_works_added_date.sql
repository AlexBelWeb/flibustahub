-- Denormalized works.added_date and catalog counters.
-- Filled here once; WarmUpCatalog refreshes after each import.
-- Missing dates are stored as '' so keyset predicates stay sargable.

ALTER TABLE works ADD COLUMN added_date TEXT NOT NULL DEFAULT '';

CREATE TEMP TABLE work_added (
  work_id INTEGER PRIMARY KEY,
  added_date TEXT NOT NULL DEFAULT ''
);

INSERT INTO work_added(work_id)
SELECT id FROM works;

UPDATE work_added SET added_date = src.added_date
  FROM (
    SELECT e.work_id, max(e.added_date) AS added_date
      FROM editions e
     WHERE e.is_active = 1 AND e.is_deleted = 0
       AND e.added_date IS NOT NULL AND trim(e.added_date) != ''
     GROUP BY e.work_id
  ) AS src
 WHERE work_added.work_id = src.work_id;

UPDATE works SET added_date = a.added_date
  FROM work_added AS a
 WHERE works.id = a.work_id;

DROP TABLE work_added;

CREATE INDEX idx_works_added ON works(added_date DESC, id DESC);
CREATE INDEX idx_series_sort ON series(sort_name, id);

CREATE TEMP TABLE listable (work_id INTEGER PRIMARY KEY);

INSERT OR IGNORE INTO listable(work_id)
SELECT DISTINCT work_id FROM editions WHERE is_active = 1 AND is_deleted = 0;

INSERT OR IGNORE INTO listable(work_id)
SELECT id FROM works
 WHERE rating IS NOT NULL
    OR (comment IS NOT NULL AND trim(comment) <> '');

INSERT INTO app_meta(key, value) VALUES ('works_total', (SELECT count(*) FROM works))
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

INSERT INTO app_meta(key, value) VALUES ('works_listable', (SELECT count(*) FROM listable))
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

DROP TABLE listable;
