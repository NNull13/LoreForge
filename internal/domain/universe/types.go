package universe

type Universe struct {
	SourcePath string
	Universe   Entity
	Rules      map[string]Entity
	Worlds     map[string]Entity
	Characters map[string]Entity
	Events     map[string]Entity
	Templates  map[string]Entity
}

type Asset struct {
	ID          string
	FileName    string
	Path        string
	MediaType   string
	Usage       string
	Description string
	Tags        []string
	Weight      int
	Optional    bool
	ModelRoles  map[string]string
}

type AssetSet struct {
	Items []Asset
}

type Entity struct {
	ID          string
	Type        string
	DisplayName string
	Summary     string
	Body        string
	Data        map[string]any
	Path        string
	Assets      AssetSet
}
