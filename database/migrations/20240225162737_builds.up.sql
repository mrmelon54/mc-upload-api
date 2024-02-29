CREATE TABLE builds
(
    id            INTEGER UNIQUE PRIMARY KEY AUTOINCREMENT,
    project       TEXT NOT NULL,
    meta          TEXT NOT NULL,
    filename      TEXT NOT NULL,
    sha512        TEXT NOT NULL,
    modrinth_id   TEXT NOT NULL,
    curseforge_id TEXT NOT NULL
);
