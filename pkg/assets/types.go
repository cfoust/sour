package assets

type IndexAsset struct {
	_    struct{} `cbor:",toarray"`
	Id   int
	Path string
}

type Asset struct {
	_    struct{} `cbor:",toarray"`
	Id   string
	Path string
}

type Mod struct {
	Id          string `cbor:"id"`
	Name        string `cbor:"name"`
	Image       string `cbor:"image"`
	Description string `cbor:"description"`
}

type GameMap struct {
	Id          string  `cbor:"id"`
	Name        string  `cbor:"name"`
	Ogz         string  `cbor:"ogz"`
	Bundle      string  `cbor:"bundle"`
	Assets      []Asset `cbor:"assets"`
	Image       string  `cbor:"image"`
	Description string  `cbor:"description"`
}

type SlimMap struct {
	Id     string
	Name   string
	Ogz    string
	Bundle string
	HasCFG bool
}

type Bundle struct {
	Id      string  `cbor:"id"`
	Desktop bool    `cbor:"desktop"`
	Web     bool    `cbor:"web"`
	Assets  []Asset `cbor:"assets"`
}

type Model struct {
	Id   string `cbor:"id"`
	Name string `cbor:"name"`
}

type Index struct {
	Assets   []string     `cbor:"assets"`
	Refs     []IndexAsset `cbor:"refs"`
	Textures []Asset      `cbor:"textures"`
	Sounds   []Asset      `cbor:"sounds"`
	Bundles  []Bundle     `cbor:"bundles"`
	Maps     []GameMap    `cbor:"maps"`
	Models   []Model      `cbor:"models"`
	Mods     []Mod        `cbor:"mods"`
}

type AssetSource struct {
	Index *Index
	Base  string
}
