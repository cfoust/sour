package min

type IndexAsset struct {
	Id   int
	Path string
}

type Index struct {
	Assets []string
	Refs   []IndexAsset
}
