package jar_parser

type FabricJson struct {
	SchemaVersion int      `json:"schemaVersion"`
	Id            string   `json:"id"`
	Version       string   `json:"version"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Authors       []string `json:"authors"`
	Contact       struct {
		Homepage string `json:"homepage"`
		Sources  string `json:"sources"`
		Issues   string `json:"issues"`
	} `json:"contact"`
	License     string `json:"license"`
	Icon        string `json:"icon"`
	Environment string `json:"environment"`
	Entrypoints struct {
		Main    []string `json:"main"`
		Modmenu []string `json:"modmenu"`
	} `json:"entrypoints"`
	Mixins  []string `json:"mixins"`
	Depends struct {
		Fabric       *FabricVersionRange `json:"fabric"`
		Minecraft    *FabricVersionRange `json:"minecraft"`
		Architectury *FabricVersionRange `json:"architectury"`
	} `json:"depends"`
	AccessWidener string `json:"accessWidener"`
}
