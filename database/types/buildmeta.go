package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

type BuildMeta struct {
	VersionNumber  string   `json:"version"`
	ReleaseChannel string   `json:"channel"`
	GameVersions   []string `json:"game_versions"`
	Loaders        []string `json:"loaders"`
	Environment    string   `json:"environment"`
}

var _ driver.Valuer = &BuildMeta{}

var _ sql.Scanner = &BuildMeta{}

func (b *BuildMeta) Value() (driver.Value, error) {
	return json.Marshal(b)
}

func (b *BuildMeta) Scan(src any) error {
	srcRaw, _ := src.(string)
	return json.Unmarshal([]byte(srcRaw), b)
}
