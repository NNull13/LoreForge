package universe

type Universe struct {
	SourcePath string
	Universe   Entity
	Artists    map[string]Artist
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

type Artist struct {
	ID           string
	Name         string
	Title        string
	Role         string
	Summary      string
	Body         string
	NonDiegetic  bool
	Voice        ArtistVoice
	Mission      ArtistMission
	Prompting    ArtistPrompting
	Presentation ArtistPresentation
	Future       ArtistFuture
	Assets       AssetSet
	Path         string
	Data         map[string]any
}

type ArtistVoice struct {
	Register    string
	Cadence     string
	Diction     string
	Stance      string
	Perspective string
	Intensity   string
}

type ArtistMission struct {
	Purpose    string
	Priorities []string
}

type ArtistPrompting struct {
	SystemIdentity string
	SystemRules    []string
	TonalBiases    []string
	LexicalCues    []string
	Forbidden      []string
}

type ArtistPresentation struct {
	Enabled         bool
	SignatureMode   string
	SignatureText   string
	FramingMode     string
	IntroTemplate   string
	OutroTemplate   string
	AllowedChannels []string
}

type ArtistFuture struct {
	MemoryMode string
}
