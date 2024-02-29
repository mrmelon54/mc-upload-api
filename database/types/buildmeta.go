package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
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
	switch srcRaw := src.(type) {
	case string:
		return json.Unmarshal([]byte(srcRaw), b)
	case []byte:
		return json.Unmarshal(srcRaw, b)
	}
	return fmt.Errorf("invalid type")
}
