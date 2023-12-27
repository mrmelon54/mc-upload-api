CREATE TABLE IF NOT EXISTS builds
(
    id       INTEGER UNIQUE PRIMARY KEY AUTOINCREMENT,
    project  TEXT NOT NULL,
    meta     BLOB NOT NULL,
    filename TEXT NOT NULL,
    sha256   TEXT NOT NULL,
    mrId     TEXT,
    cfId     TEXT
);
