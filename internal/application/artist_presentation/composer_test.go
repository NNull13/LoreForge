package artist_presentation

import (
	"testing"
	"time"

	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
)

func TestComposeAppliesTextFramingAndSignature(t *testing.T) {
	t.Parallel()

	item, applied := Compose(publication.Item{
		EpisodeID: "ep-1",
		Content:   "Aria hears the gate answer.",
		CreatedAt: time.Now().UTC(),
	}, episode.ArtistLens{
		ID:   "ash-chorister",
		Name: "The Ash Chorister",
		Presentation: episode.ArtistPresentationSnapshot{
			Enabled:       true,
			SignatureMode: "append",
			FramingMode:   "intro_outro",
			IntroTemplate: "From the archive:",
			OutroTemplate: "End of record.",
			SignatureText: "Filed by The Ash Chorister.",
		},
	}, publication.ChannelFilesystem)

	if applied.SignatureApplied == "" {
		t.Fatal("expected signature to be applied")
	}
	if got := item.Content; got != "From the archive:\n\nAria hears the gate answer.\n\nEnd of record.\n\nFiled by The Ash Chorister." {
		t.Fatalf("unexpected composed content: %q", got)
	}
}

func TestComposeAppliesThreadAndAssetCaption(t *testing.T) {
	t.Parallel()

	thread, _ := Compose(publication.Item{
		EpisodeID: "ep-2",
		Parts:     []string{"one", "two"},
		Content:   "one\n\ntwo",
		CreatedAt: time.Now().UTC(),
	}, episode.ArtistLens{
		ID:   "ash-chorister",
		Name: "The Ash Chorister",
		Presentation: episode.ArtistPresentationSnapshot{
			Enabled:       true,
			SignatureMode: "append",
			FramingMode:   "intro",
			IntroTemplate: "Intro",
			SignatureText: "Signature",
		},
	}, publication.ChannelFilesystem)
	if thread.Parts[0] != "Intro one" || thread.Parts[1] != "two Signature" {
		t.Fatalf("unexpected thread parts: %#v", thread.Parts)
	}

	asset, _ := Compose(publication.Item{
		EpisodeID: "ep-3",
		AssetPath: "/tmp/image.png",
		CreatedAt: time.Now().UTC(),
	}, episode.ArtistLens{
		ID:   "ash-chorister",
		Name: "The Ash Chorister",
		Presentation: episode.ArtistPresentationSnapshot{
			Enabled:         true,
			SignatureMode:   "presentation_only",
			FramingMode:     "intro",
			IntroTemplate:   "Asset intro",
			AllowedChannels: []string{"filesystem"},
		},
	}, publication.ChannelFilesystem)
	if asset.Caption == "" {
		t.Fatal("expected caption for asset publication")
	}
}

func TestComposeSkipsDisallowedChannel(t *testing.T) {
	t.Parallel()

	item, applied := Compose(publication.Item{
		EpisodeID: "ep-4",
		Content:   "No change",
		CreatedAt: time.Now().UTC(),
	}, episode.ArtistLens{
		ID:   "ash-chorister",
		Name: "The Ash Chorister",
		Presentation: episode.ArtistPresentationSnapshot{
			Enabled:         true,
			AllowedChannels: []string{"filesystem"},
		},
	}, publication.ChannelTwitter)
	if item.Content != "No change" || !applied.Enabled {
		t.Fatalf("unexpected disallowed channel result: %#v %#v", item, applied)
	}
}
