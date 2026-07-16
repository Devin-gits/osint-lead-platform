// Package util holds small, dependency-free helpers used across the control plane.
package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// NewID generates a random UUID v4 string without adding a third-party dependency.
func NewID() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		// crypto/rand should never fail on a healthy system; if it does, panic.
		panic(fmt.Sprintf("failed to read random bytes: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
