package main

import "github.com/MrMelon54/mc-upload-api/uploader"

type Config struct {
	Listen     string                    `yaml:"listen"`
	Login      LoginConfig               `yaml:"login"`
	Modrinth   uploader.ModrinthConfig   `yaml:"modrinth"`
	Curseforge uploader.CurseforgeConfig `yaml:"curseforge"`
}

type LoginConfig struct {
	Url   string `yaml:"url"`
	Owner string `yaml:"owner"`
}
