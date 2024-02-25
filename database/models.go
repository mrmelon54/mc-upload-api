// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package database

import (
	"database/sql"

	"github.com/mrmelon54/mc-upload-api/database/types"
)

type Build struct {
	ID           int64
	Project      string
	Meta         *types.BuildMeta
	Filename     string
	Sha512       string
	ModrinthID   sql.NullString
	CurseforgeID sql.NullString
}