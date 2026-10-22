PRAGMA foreign_keys = ON;

CREATE TABLE report_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    rules_version INTEGER NOT NULL CHECK (rules_version > 0),
    host TEXT NOT NULL,
    timezone TEXT NOT NULL,
    config_json TEXT NOT NULL,
    config_hash BLOB NOT NULL CHECK (length(config_hash) = 32),
    visitor_key_id TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE log_generations (
    id INTEGER PRIMARY KEY,
    committed_offset INTEGER NOT NULL DEFAULT 0 CHECK (committed_offset >= 0),
    committed_sha256 BLOB,
    head_length INTEGER NOT NULL DEFAULT 0 CHECK (head_length >= 0),
    tail_length INTEGER NOT NULL DEFAULT 0 CHECK (tail_length >= 0),
    head_sha256 BLOB,
    tail_sha256 BLOB,
    sealed INTEGER NOT NULL DEFAULT 0 CHECK (sealed IN (0, 1)),
    first_ts_us INTEGER,
    last_ts_us INTEGER,
    valid_lines INTEGER NOT NULL DEFAULT 0 CHECK (valid_lines >= 0),
    invalid_lines INTEGER NOT NULL DEFAULT 0 CHECK (invalid_lines >= 0),
    host_out_lines INTEGER NOT NULL DEFAULT 0 CHECK (host_out_lines >= 0),
    article_access_lines INTEGER NOT NULL DEFAULT 0 CHECK (article_access_lines >= 0),
    article_excluded_lines INTEGER NOT NULL DEFAULT 0 CHECK (article_excluded_lines >= 0),
    article_other_lines INTEGER NOT NULL DEFAULT 0 CHECK (article_other_lines >= 0),
    unidentified_article_lines INTEGER NOT NULL DEFAULT 0 CHECK (unidentified_article_lines >= 0),
    oversized_dimension_lines INTEGER NOT NULL DEFAULT 0 CHECK (oversized_dimension_lines >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((committed_offset = 0 AND committed_sha256 IS NULL) OR
           (committed_offset > 0 AND committed_sha256 IS NOT NULL AND length(committed_sha256) = 32)),
    CHECK (head_length <= committed_offset AND tail_length <= committed_offset),
    CHECK ((head_length = 0 AND head_sha256 IS NULL) OR
           (head_length > 0 AND head_sha256 IS NOT NULL AND length(head_sha256) = 32)),
    CHECK ((tail_length = 0 AND tail_sha256 IS NULL) OR
           (tail_length > 0 AND tail_sha256 IS NOT NULL AND length(tail_sha256) = 32)),
    CHECK ((first_ts_us IS NULL AND last_ts_us IS NULL) OR
           (first_ts_us IS NOT NULL AND last_ts_us IS NOT NULL AND first_ts_us <= last_ts_us)),
    CHECK (valid_lines = host_out_lines + article_access_lines + article_excluded_lines + article_other_lines),
    CHECK (unidentified_article_lines <= article_access_lines)
);

CREATE TABLE log_files (
    id INTEGER PRIMARY KEY,
    generation_id INTEGER NOT NULL REFERENCES log_generations(id),
    path TEXT NOT NULL,
    compression TEXT NOT NULL CHECK (compression IN ('none', 'gzip')),
    device TEXT,
    inode TEXT,
    observed_size INTEGER NOT NULL CHECK (observed_size >= 0),
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);
CREATE INDEX log_files_identity ON log_files(device, inode);
CREATE INDEX log_files_path ON log_files(path);
CREATE INDEX log_files_generation ON log_files(generation_id);

CREATE TABLE articles (
    article_id INTEGER PRIMARY KEY CHECK (article_id > 0),
    path TEXT NOT NULL UNIQUE
);

CREATE TABLE article_daily (
    day TEXT NOT NULL,
    article_id INTEGER NOT NULL REFERENCES articles(article_id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    unidentified_count INTEGER NOT NULL DEFAULT 0 CHECK (unidentified_count >= 0 AND unidentified_count <= access_count),
    PRIMARY KEY (day, article_id)
) WITHOUT ROWID;

CREATE TABLE article_daily_visitors (
    day TEXT NOT NULL,
    article_id INTEGER NOT NULL,
    visitor_key BLOB NOT NULL CHECK (length(visitor_key) = 32),
    PRIMARY KEY (day, article_id, visitor_key),
    FOREIGN KEY (day, article_id) REFERENCES article_daily(day, article_id)
) WITHOUT ROWID;

CREATE TABLE crawler_agents (
    id INTEGER PRIMARY KEY,
    rule_id TEXT NOT NULL,
    ua_hash BLOB NOT NULL UNIQUE CHECK (length(ua_hash) = 32),
    display_ua TEXT NOT NULL CHECK (length(display_ua) <= 1024),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1))
);
CREATE TABLE crawler_daily (
    day TEXT NOT NULL,
    agent_id INTEGER NOT NULL REFERENCES crawler_agents(id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    PRIMARY KEY (day, agent_id)
) WITHOUT ROWID;

CREATE TABLE not_found_paths (
    id INTEGER PRIMARY KEY,
    path_hash BLOB NOT NULL UNIQUE CHECK (length(path_hash) = 32),
    display_path TEXT NOT NULL CHECK (length(display_path) <= 2048),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1))
);
CREATE TABLE not_found_daily (
    day TEXT NOT NULL,
    path_id INTEGER NOT NULL REFERENCES not_found_paths(id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    PRIMARY KEY (day, path_id)
) WITHOUT ROWID;

CREATE TABLE scan_sources (
    id INTEGER PRIMARY KEY,
    ip TEXT NOT NULL UNIQUE
);
CREATE TABLE scan_daily_paths (
    day TEXT NOT NULL,
    source_id INTEGER NOT NULL REFERENCES scan_sources(id),
    path_id INTEGER NOT NULL REFERENCES not_found_paths(id),
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    first_ts_us INTEGER NOT NULL,
    last_ts_us INTEGER NOT NULL CHECK (last_ts_us >= first_ts_us),
    PRIMARY KEY (day, source_id, path_id)
) WITHOUT ROWID;
CREATE INDEX scan_daily_paths_source ON scan_daily_paths(source_id, day);
CREATE TABLE scan_daily_agents (
    day TEXT NOT NULL,
    source_id INTEGER NOT NULL REFERENCES scan_sources(id),
    ua_hash BLOB NOT NULL CHECK (length(ua_hash) = 32),
    display_ua TEXT NOT NULL CHECK (length(display_ua) <= 1024),
    rule_id TEXT NOT NULL,
    access_count INTEGER NOT NULL CHECK (access_count > 0),
    PRIMARY KEY (day, source_id, ua_hash)
) WITHOUT ROWID;
CREATE INDEX scan_daily_agents_source ON scan_daily_agents(source_id, day);
