package jar_parser

type ForgeToml struct {
	ModLoader       string `toml:"modLoader"`
	LoaderVersion   string `toml:"loaderVersion"`
	IssueTrackerURL string `toml:"issueTrackerURL"`
	License         string `toml:"license"`
	Mods            []struct {
		ModID       string `toml:"modId"`
		Version     string `toml:"version"`
		DisplayName string `toml:"displayName"`
		Authors     string `toml:"authors"`
		Description string `toml:"description"`
		LogoFile    string `toml:"logoFile"`
	} `toml:"mods"`
	Dependencies map[string][]struct {
		ModID        string `toml:"modId"`
		Mandatory    bool   `toml:"mandatory"`
		VersionRange string `toml:"versionRange"`
		Ordering     string `toml:"ordering"`
		Side         string `toml:"side"`
	} `toml:"dependencies"`
}
