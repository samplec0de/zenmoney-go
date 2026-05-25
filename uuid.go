package zenmoney

import (
	"crypto/rand"
	"fmt"
)

// NewUUID returns a random UUID v4 string (lowercase, hyphenated, 36 chars).
// Used as a client-side ID generator for new entities (transactions, tags,
// merchants) before they are pushed to the server.
func NewUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("uuid: read random: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
