package util

import (
	"crypto/rand"
	"encoding/hex"
)

func NewEpisodeID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "episode-unknown"
	}
	return "ep-" + hex.EncodeToString(b)
}
