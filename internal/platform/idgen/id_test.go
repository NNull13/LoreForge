package idgen

import (
	"strings"
	"testing"
)

func TestCryptoIDGeneratorNewEpisodeID(t *testing.T) {
	t.Parallel()

	id := CryptoIDGenerator{}.NewEpisodeID()
	if !strings.HasPrefix(id, "ep-") {
		t.Fatalf("unexpected prefix: %s", id)
	}
	if len(id) != 19 {
		t.Fatalf("unexpected id length: %d", len(id))
	}
}
