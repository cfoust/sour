package catalog

type MapEntry struct {
	Author      string   `json:"author,omitempty"`
	Date        string   `json:"date,omitempty"`
	Description string   `json:"description,omitempty"`
	Image       string   `json:"image,omitempty"`
	Gif         string   `json:"gif,omitempty"`
	Images      []string `json:"images,omitempty"`
	Modes       []string `json:"modes,omitempty"`
	Players     string   `json:"players,omitempty"`
}

type ModEntry struct {
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
}

type Catalog struct {
	Maps map[string]MapEntry `json:"maps,omitempty"`
	Mods map[string]ModEntry `json:"mods,omitempty"`
}

// ResolvedMapEntry has image/gif hashes replaced with full URLs.
type ResolvedMapEntry struct {
	Author      string   `json:"author,omitempty"`
	Date        string   `json:"date,omitempty"`
	Description string   `json:"description,omitempty"`
	ImageURL    string   `json:"imageUrl,omitempty"`
	GifURL      string   `json:"gifUrl,omitempty"`
	ImageURLs   []string `json:"imageUrls,omitempty"`
	Modes       []string `json:"modes,omitempty"`
	Players     string   `json:"players,omitempty"`
}

// ResolvedModEntry has image hash replaced with a full URL.
type ResolvedModEntry struct {
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

// ResolvedCatalog is the merged catalog with all image/gif hashes resolved to URLs.
type ResolvedCatalog struct {
	Maps map[string]ResolvedMapEntry `json:"maps"`
	Mods map[string]ResolvedModEntry `json:"mods,omitempty"`
}
