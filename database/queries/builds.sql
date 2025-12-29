-- name: CreateBuild :execlastid
INSERT INTO builds (project, meta, filename, sha512, modrinth_id, curseforge_id)
VALUES (?, ?, ?, ?, "", "");

-- name: UpdateModrinthFile :exec
UPDATE builds
SET modrinth_id = ?
WHERE id = ?;

-- name: UpdateCurseforgeFile :exec
UPDATE builds
SET curseforge_id = ?
WHERE id = ?;

-- name: ListBuilds :many
SELECT meta, filename, sha512, modrinth_id, curseforge_id
FROM builds
WHERE project = ?
ORDER BY id;

-- name: HashExists :one
SELECT EXISTS(SELECT 1 FROM builds WHERE sha512 = ?);
