-- Catalog schema. FTS CREATE statements in this file are the single source
-- for migrate and rebuildFts; do not duplicate them elsewhere.

CREATE TABLE works (
  id                    INTEGER PRIMARY KEY,
  work_key              TEXT    NOT NULL UNIQUE,
  title                 TEXT    NOT NULL,
  sort_title            TEXT    NOT NULL,
  authors_text          TEXT    NOT NULL,
  lang                  TEXT,
  annotation            TEXT,
  annotation_checked_at TEXT,
  rating                INTEGER,
  rating_updated_at     TEXT,
  comment               TEXT,
  comment_updated_at    TEXT,
  exported_at           TEXT,
  created_at            TEXT    NOT NULL,
  updated_at            TEXT    NOT NULL
);

CREATE TABLE editions (
  id           INTEGER PRIMARY KEY,
  libid        TEXT    NOT NULL UNIQUE,
  work_id      INTEGER NOT NULL REFERENCES works(id) ON DELETE CASCADE,
  archive_name TEXT    NOT NULL,
  file_name    TEXT    NOT NULL,
  file_ext     TEXT    NOT NULL DEFAULT 'fb2',
  size         INTEGER,
  series       TEXT,
  series_no    TEXT,
  lang         TEXT,
  librate      INTEGER,
  keywords     TEXT,
  added_date   TEXT,
  is_deleted   INTEGER NOT NULL DEFAULT 0,
  is_active    INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE authors (
  id           INTEGER PRIMARY KEY,
  author_key   TEXT NOT NULL UNIQUE,
  last_name    TEXT NOT NULL,
  first_name   TEXT NOT NULL,
  middle_name  TEXT NOT NULL,
  display_name TEXT NOT NULL,
  sort_name    TEXT NOT NULL,
  work_count   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE work_authors (
  work_id   INTEGER NOT NULL REFERENCES works(id)   ON DELETE CASCADE,
  author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
  position  INTEGER NOT NULL,
  PRIMARY KEY (work_id, author_id)
) WITHOUT ROWID;

CREATE TABLE genres (
  id         INTEGER PRIMARY KEY,
  code       TEXT NOT NULL UNIQUE,
  name_ru    TEXT NOT NULL,
  work_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE edition_genres (
  edition_id INTEGER NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
  genre_id   INTEGER NOT NULL REFERENCES genres(id)   ON DELETE CASCADE,
  PRIMARY KEY (edition_id, genre_id)
) WITHOUT ROWID;

CREATE TABLE work_genres (
  work_id  INTEGER NOT NULL,
  genre_id INTEGER NOT NULL,
  PRIMARY KEY (work_id, genre_id)
) WITHOUT ROWID;

CREATE TABLE series (
  id         INTEGER PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE,
  sort_name  TEXT NOT NULL,
  work_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE recently_viewed (
  work_id   INTEGER PRIMARY KEY REFERENCES works(id) ON DELETE CASCADE,
  viewed_at TEXT NOT NULL
);

CREATE TABLE search_history (
  id          INTEGER PRIMARY KEY,
  query       TEXT NOT NULL,
  searched_at TEXT NOT NULL
);

CREATE TABLE ai_recommendations (
  id            INTEGER PRIMARY KEY,
  provider      TEXT NOT NULL,
  model         TEXT NOT NULL,
  prompt_hash   TEXT NOT NULL,
  response_json TEXT NOT NULL,
  created_at    TEXT NOT NULL,
  UNIQUE (provider, model, prompt_hash)
);

CREATE TABLE import_batches (
  id                     INTEGER PRIMARY KEY,
  started_at             TEXT NOT NULL,
  finished_at            TEXT,
  status                 TEXT NOT NULL,
  inpx_path              TEXT NOT NULL,
  inpx_version           TEXT,
  records_seen           INTEGER NOT NULL DEFAULT 0,
  works_added            INTEGER NOT NULL DEFAULT 0,
  editions_added         INTEGER NOT NULL DEFAULT 0,
  editions_updated       INTEGER NOT NULL DEFAULT 0,
  editions_deactivated   INTEGER NOT NULL DEFAULT 0,
  libid_collisions       INTEGER NOT NULL DEFAULT 0,
  notes                  TEXT
);

CREATE TABLE app_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE INDEX idx_editions_work        ON editions(work_id);
CREATE INDEX idx_editions_active      ON editions(is_active, is_deleted);
CREATE INDEX idx_editions_added       ON editions(added_date DESC, id DESC);
CREATE INDEX idx_editions_archive     ON editions(archive_name);
CREATE INDEX idx_editions_series      ON editions(series);
CREATE INDEX idx_works_sort_title     ON works(sort_title, id);
CREATE INDEX idx_works_rating         ON works(rating DESC, rating_updated_at DESC) WHERE rating IS NOT NULL;
CREATE INDEX idx_authors_sort         ON authors(sort_name, id);
CREATE INDEX idx_authors_work_count   ON authors(work_count DESC, id);
CREATE INDEX idx_work_authors_author  ON work_authors(author_id);
CREATE INDEX idx_work_genres_genre    ON work_genres(genre_id, work_id);
CREATE INDEX idx_recently_viewed_time ON recently_viewed(viewed_at DESC);
CREATE INDEX idx_search_history_time  ON search_history(searched_at DESC);

CREATE VIRTUAL TABLE works_fts   USING fts5(title, authors, series, tokenize = 'unicode61 remove_diacritics 2');
CREATE VIRTUAL TABLE authors_fts USING fts5(display_name, sort_name, tokenize = 'unicode61 remove_diacritics 2');
CREATE VIRTUAL TABLE series_fts  USING fts5(name, tokenize = 'unicode61 remove_diacritics 2');

CREATE TRIGGER works_fts_au AFTER UPDATE OF title, authors_text ON works BEGIN
  DELETE FROM works_fts WHERE rowid = old.id;
  INSERT INTO works_fts(rowid, title, authors, series)
  SELECT new.id,
         normalize(new.title),
         normalize(new.authors_text),
         (SELECT normalize(group_concat(DISTINCT e.series))
            FROM editions e
           WHERE e.work_id = new.id
             AND e.is_active = 1 AND e.is_deleted = 0
             AND e.series IS NOT NULL AND trim(e.series) != '');
END;

CREATE TRIGGER works_fts_ad AFTER DELETE ON works BEGIN
  DELETE FROM works_fts WHERE rowid = old.id;
END;

CREATE TRIGGER authors_fts_au AFTER UPDATE OF display_name, sort_name ON authors BEGIN
  DELETE FROM authors_fts WHERE rowid = old.id;
  INSERT INTO authors_fts(rowid, display_name, sort_name)
  VALUES (new.id, normalize(new.display_name), normalize(new.sort_name));
END;

CREATE TRIGGER authors_fts_ad AFTER DELETE ON authors BEGIN
  DELETE FROM authors_fts WHERE rowid = old.id;
END;
