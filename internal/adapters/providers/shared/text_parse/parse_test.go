package text_parse

import "testing"

func TestParseStructuredContentParts(t *testing.T) {
	t.Parallel()

	content, parts, title, metadata, err := ParseStructuredContent(`{"title":"Song","parts":["one","two"]}`)
	if err != nil {
		t.Fatalf("ParseStructuredContent returned error: %v", err)
	}
	if content != "one\n\ntwo" {
		t.Fatalf("content = %q", content)
	}
	if len(parts) != 2 || title != "Song" || metadata["title"] != "Song" {
		t.Fatalf("unexpected parse result: %#v %#v %q", parts, metadata, title)
	}
}

func TestParseStructuredContentBodyAndErrors(t *testing.T) {
	t.Parallel()

	content, parts, _, _, err := ParseStructuredContent(`{"body":"hello"}`)
	if err != nil {
		t.Fatalf("ParseStructuredContent returned error: %v", err)
	}
	if content != "hello" || parts != nil {
		t.Fatalf("unexpected body parse result: %q %#v", content, parts)
	}
	if _, _, _, _, err := ParseStructuredContent(`{"parts":[1]}`); err == nil {
		t.Fatal("expected invalid parts error")
	}
}
