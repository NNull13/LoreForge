package artist_presentation

import (
	"fmt"
	"strings"

	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
)

type Applied struct {
	Channel          string   `json:"channel"`
	Enabled          bool     `json:"enabled"`
	SignatureMode    string   `json:"signature_mode,omitempty"`
	FramingMode      string   `json:"framing_mode,omitempty"`
	IntroApplied     string   `json:"intro_applied,omitempty"`
	OutroApplied     string   `json:"outro_applied,omitempty"`
	SignatureApplied string   `json:"signature_applied,omitempty"`
	AllowedChannels  []string `json:"allowed_channels,omitempty"`
}

func Compose(item publication.Item, artist episode.ArtistLens, channel publication.ChannelName) (publication.Item, Applied) {
	applied := Applied{
		Channel:         string(channel),
		Enabled:         artist.Presentation.Enabled,
		SignatureMode:   artist.Presentation.SignatureMode,
		FramingMode:     artist.Presentation.FramingMode,
		AllowedChannels: append([]string(nil), artist.Presentation.AllowedChannels...),
	}
	if !artist.Presentation.Enabled {
		return item, applied
	}
	if len(artist.Presentation.AllowedChannels) > 0 && !contains(artist.Presentation.AllowedChannels, string(channel)) {
		return item, applied
	}
	intro := strings.TrimSpace(artist.Presentation.IntroTemplate)
	outro := strings.TrimSpace(artist.Presentation.OutroTemplate)
	signature := strings.TrimSpace(artist.Presentation.SignatureText)
	if signature == "" {
		signature = fmt.Sprintf("Filed by %s.", coalesce(artist.Name, artist.ID))
	}
	if item.AssetPath != "" {
		item.Caption = joinParagraphs(framedText("", intro, outro, artist.Presentation.FramingMode), signatureForMode(signature, artist.Presentation.SignatureMode))
		applied.IntroApplied = intro
		applied.OutroApplied = outro
		applied.SignatureApplied = signatureForMode(signature, artist.Presentation.SignatureMode)
		return item, applied
	}
	if len(item.Parts) > 0 {
		item.Parts = applyThreadPresentation(item.Parts, intro, outro, signature, artist.Presentation.FramingMode, artist.Presentation.SignatureMode)
		item.Content = strings.Join(item.Parts, "\n\n")
		applied.IntroApplied = intro
		applied.OutroApplied = outro
		applied.SignatureApplied = signatureForMode(signature, artist.Presentation.SignatureMode)
		return item, applied
	}
	item.Content = applyTextPresentation(item.Content, intro, outro, signature, artist.Presentation.FramingMode, artist.Presentation.SignatureMode)
	applied.IntroApplied = intro
	applied.OutroApplied = outro
	applied.SignatureApplied = signatureForMode(signature, artist.Presentation.SignatureMode)
	return item, applied
}

func applyTextPresentation(content, intro, outro, signature, framingMode, signatureMode string) string {
	out := framedText(content, intro, outro, framingMode)
	switch signatureMode {
	case "append":
		out = joinParagraphs(out, signature)
	case "prepend":
		out = joinParagraphs(signature, out)
	}
	return out
}

func framedText(content, intro, outro, framingMode string) string {
	out := strings.TrimSpace(content)
	switch framingMode {
	case "intro":
		out = joinParagraphs(intro, out)
	case "outro":
		out = joinParagraphs(out, outro)
	case "intro_outro":
		out = joinParagraphs(intro, out, outro)
	}
	return out
}

func applyThreadPresentation(parts []string, intro, outro, signature, framingMode, signatureMode string) []string {
	if len(parts) == 0 {
		return nil
	}
	out := append([]string(nil), parts...)
	switch framingMode {
	case "intro":
		out[0] = joinInline(intro, out[0])
	case "outro":
		out[len(out)-1] = joinInline(out[len(out)-1], outro)
	case "intro_outro":
		out[0] = joinInline(intro, out[0])
		out[len(out)-1] = joinInline(out[len(out)-1], outro)
	}
	switch signatureMode {
	case "append":
		out[len(out)-1] = joinInline(out[len(out)-1], signature)
	case "prepend":
		out[0] = joinInline(signature, out[0])
	}
	return out
}

func signatureForMode(signature, mode string) string {
	switch mode {
	case "presentation_only", "append", "prepend":
		return signature
	default:
		return ""
	}
}

func joinParagraphs(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "\n\n")
}

func joinInline(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " ")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
