package main

import "github.com/mrmelon54/mc-upload-api/uploader"

type Config struct {
	Listen     string                    `yaml:"listen"`
	Modrinth   uploader.ModrinthConfig   `yaml:"modrinth"`
	Curseforge uploader.CurseforgeConfig `yaml:"curseforge"`
}
