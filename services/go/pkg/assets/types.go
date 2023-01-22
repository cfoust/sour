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
	Id          string
	Name        string
	Image       string
	Description string
}

type GameMap struct {
	Id          string
	Name        string
	Ogz         int
	Bundle      string
	Assets      []Asset
	Image       string
	Description string
}

type Bundle struct {
	Id      string
	Desktop bool
	Web     bool
	Assets  []Asset
}

type Model struct {
	Id   string
	Name string
}

type Index struct {
	Assets   []string
	Refs     []IndexAsset
	Textures []Asset
	Sounds   []Asset
	Bundles  []Bundle
	Maps     []GameMap
	Models   []Model
	Mods     []Mod
}

type AssetSource struct {
	Index *Index
	Base  string
}
