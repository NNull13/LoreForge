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

type Entity struct {
	ID          string
	Type        string
	DisplayName string
	Summary     string
	Body        string
	Data        map[string]any
	Path        string
}
