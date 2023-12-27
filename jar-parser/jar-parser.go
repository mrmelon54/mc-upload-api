package jar_parser

import (
	"archive/zip"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"io"
)

type ModMetadata struct {
	VersionNumber  string
	ReleaseChannel string
	GameVersions   []*semver.Constraints
	Loaders        []string
	Environment    string
}

func JarParser(r io.ReaderAt, size int64) (ModMetadata, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return ModMetadata{}, err
	}

	meta := ModMetadata{
		ReleaseChannel: "release",
	}
	for _, i := range zr.File {
		switch i.Name {
		case "fabric.mod.json":
			var fabricJson FabricJson
			open, err := i.Open()
			if err != nil {
				return ModMetadata{}, err
			}
			if err := json.NewDecoder(open).Decode(&fabricJson); err != nil {
				return ModMetadata{}, err
			}
			meta.VersionNumber = fabricJson.Version
			meta.Loaders = append(meta.Loaders, "fabric")
			meta.Environment = fabricJson.Environment
			constraint, err := semver.NewConstraint(fabricJson.Depends.Minecraft)
			if err != nil {
				return ModMetadata{}, err
			}
			meta.GameVersions = append(meta.GameVersions, constraint)
		case "quilt.mod.json":
			var quiltJson QuiltJson
			open, err := i.Open()
			if err != nil {
				return ModMetadata{}, err
			}
			if err := json.NewDecoder(open).Decode(&quiltJson); err != nil {
				return ModMetadata{}, err
			}
			meta.VersionNumber = quiltJson.QuiltLoader.Version
			meta.Loaders = append(meta.Loaders, "quilt")
			meta.Environment = quiltJson.Minecraft.Environment
			for _, j := range quiltJson.QuiltLoader.Depends {
				if j.Id == "minecraft" {
					constraint, err := semver.NewConstraint(j.Version)
					if err != nil {
						return ModMetadata{}, err
					}
					meta.GameVersions = append(meta.GameVersions, constraint)
				}
			}
		case "META-INF/mods.toml":
			var forgeToml ForgeToml
			open, err := i.Open()
			if err != nil {
				return ModMetadata{}, err
			}
			if _, err := toml.NewDecoder(open).Decode(&forgeToml); err != nil {
				return ModMetadata{}, err
			}
			var modId string
			for _, j := range forgeToml.Mods {
				modId = j.ModID
				meta.VersionNumber = j.Version
			}
			meta.Loaders = append(meta.Loaders, "forge")
			for k, v := range forgeToml.Dependencies {
				if k == modId {
					for _, j := range v {
						if j.ModID == "minecraft" {
							versionRange, err := ForgeVersionRange(j.VersionRange)
							if err != nil {
								return ModMetadata{}, err
							}
							meta.GameVersions = append(meta.GameVersions, versionRange)
						}
					}
				}
			}
		}
	}
	return meta, nil
}
