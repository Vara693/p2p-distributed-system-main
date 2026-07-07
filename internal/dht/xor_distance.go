package dht

import (
	"encoding/hex"
	"errors"
	"math/big"
)

const keyBytes = 32

// KeyFromHex decodes a 64-char hex string (SHA-256) into 32 bytes.
func KeyFromHex(s string) ([]byte, error) {
	if len(s) != keyBytes*2 {
		return nil, errors.New("invalid key length")
	}
	return hex.DecodeString(s)
}

// XOR returns a XOR b (same length).
func XOR(a, b []byte) []byte {
	if len(a) != len(b) {
		panic("xor length mismatch")
	}
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// Distance interprets XOR(a,b) as a big integer (for ordering).
func Distance(a, b []byte) *big.Int {
	x := XOR(a, b)
	return new(big.Int).SetBytes(x)
}

// Closer returns true if a is closer to target than b (smaller XOR distance to target).
func Closer(target, a, b []byte) bool {
	da := Distance(target, a)
	db := Distance(target, b)
	return da.Cmp(db) < 0
}
