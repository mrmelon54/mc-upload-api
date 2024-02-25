-- name: CreateBuild :execlastid
INSERT INTO builds (project, meta, filename, sha512)
VALUES (?, ?, ?, ?);

-- name: UpdateModrinthFile :exec
UPDATE builds
SET mrId = ?
WHERE id = ?;

-- name: UpdateCurseforgeFile :exec
UPDATE builds
SET cfId = ?
WHERE id = ?;

-- name: ListBuilds :many
SELECT meta, filename, sha512, mrId, cfId
FROM builds
WHERE project = ?
ORDER BY id;
