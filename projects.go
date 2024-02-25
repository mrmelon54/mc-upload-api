package mc_upload_api

type ProjectsConfig map[string]Project

type Project struct {
	ProjectDetails `yaml:",inline"`
	Token          string `yaml:"token"`
}

type ProjectDetails struct {
	Name       string          `yaml:"name" json:"name"`
	Modrinth   ProjectPlatform `yaml:"modrinth" json:"modrinth"`
	Curseforge ProjectPlatform `yaml:"curseforge" json:"curseforge"`
	Github     string          `yaml:"github" json:"github"`
}

type ProjectPlatform struct {
	Url string `yaml:"url" json:"url"`
	Id  string `yaml:"id" json:"id"`
}

func (p ProjectPlatform) Enabled() bool {
	return p.Id != ""
}
