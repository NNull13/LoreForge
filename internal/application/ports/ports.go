package ports

import (
	"context"
	"time"

	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
	"loreforge/internal/domain/scheduling"
	"loreforge/internal/domain/universe"
)

type Generator interface {
	ID() string
	Type() episode.OutputType
	Generate(ctx context.Context, brief episode.Brief, state episode.State) (episode.Output, error)
}

type GeneratorConfig struct {
	ID                    string
	ProfileID             string
	Type                  episode.OutputType
	Style                 string
	SchedulerEnabled      bool
	PublishTargets        []publication.ChannelName
	Scheduler             scheduling.Config
	Seed                  int64
	ProviderDriver        string
	ProviderModel         string
	ProviderConfig        map[string]any
	Options               map[string]any
	ReferenceMode         string
	ContinuityScope       string
	MaxContinuityItems    int
	MaxAssetReferences    int
	IncludeTextMemories   bool
	AssetUsageAllowlist   []string
	TextConstraints       *episode.TextConstraints
	PromptOverrides       map[string]any
	PresentationOverrides map[string]any
}

type RegisteredGenerator struct {
	Generator Generator
	Config    GeneratorConfig
}

type GeneratorRegistry interface {
	GetByID(id string) (RegisteredGenerator, bool)
	GetByType(outputType episode.OutputType) (RegisteredGenerator, bool)
	List() []RegisteredGenerator
}

type UniverseRepository interface {
	Load(ctx context.Context) (universe.Universe, error)
}

type EpisodeRepository interface {
	Save(ctx context.Context, record episode.Record) (episode.StoredRecord, error)
	FindByID(ctx context.Context, id string) (episode.StoredRecord, error)
	RecentCombos(ctx context.Context, limit int) ([]episode.Combo, error)
	RecentCombosByGenerator(ctx context.Context, generatorID string, limit int) ([]episode.Combo, error)
	RecentReferencesByGenerator(ctx context.Context, generatorID string, limit int) ([]episode.ContinuityReference, error)
}

type SchedulerStateRepository interface {
	Load(ctx context.Context, generatorID string) (scheduling.State, error)
	Save(ctx context.Context, generatorID string, state scheduling.State) error
	ListGeneratorIDs(ctx context.Context) ([]string, error)
}

type Publisher interface {
	Name() publication.ChannelName
	Publish(ctx context.Context, item publication.Item) (publication.Result, error)
}

type PublisherRegistry interface {
	Get(name publication.ChannelName) (Publisher, bool)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewEpisodeID() string
}

type Hasher interface {
	Hash(ctx context.Context) (string, error)
}
