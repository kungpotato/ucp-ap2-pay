package main

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
