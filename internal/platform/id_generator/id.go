package id_generator

import (
	"crypto/rand"
	"encoding/hex"
)

type CryptoIDGenerator struct{}

func (CryptoIDGenerator) NewEpisodeID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "episode-unknown"
	}
	return "ep-" + hex.EncodeToString(buf)
}
