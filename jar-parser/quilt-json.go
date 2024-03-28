package jar_parser

type QuiltJson struct {
	SchemaVersion int      `json:"schema_version"`
	Mixin         []string `json:"mixin"`
	QuiltLoader   struct {
		Group    string `json:"group"`
		Id       string `json:"id"`
		Version  string `json:"version"`
		Metadata struct {
			Name         string            `json:"name"`
			Description  string            `json:"description"`
			Contributors map[string]string `json:"contributors"`
			Contact      struct {
				Homepage string `json:"homepage"`
				Sources  string `json:"sources"`
				Issues   string `json:"issues"`
			} `json:"contact"`
			License string `json:"license"`
			Icon    string `json:"icon"`
		} `json:"metadata"`
		IntermediateMappings string `json:"intermediate_mappings"`
		Entrypoints          struct {
			Init    []string `json:"init"`
			Modmenu []string `json:"modmenu"`
		} `json:"entrypoints"`
		Depends []struct {
			Id      string              `json:"id"`
			Version *FabricVersionRange `json:"version"`
		} `json:"depends"`
	} `json:"quilt_loader"`
	Minecraft struct {
		Environment string `json:"environment"`
	} `json:"minecraft"`
	AccessWidener string `json:"access_widener"`
}
