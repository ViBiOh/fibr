package thumbnail

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
)

// StreamHasher is a hasher encapsulation
type StreamHasher struct {
	hasher hash.Hash
}

// Stream create a new stream hasher
func Stream() StreamHasher {
	return StreamHasher{
		hasher: sha1.New(),
	}
}

// Write writes content to the hasher
func (s StreamHasher) Write(o interface{}) StreamHasher {
	// no err check https://golang.org/pkg/hash/#Hash
	_, _ = fmt.Fprintf(s.hasher, "%#v", o)

	return s
}

// Sum returns the result of hashing
func (s StreamHasher) Sum() string {
	return hex.EncodeToString(s.hasher.Sum(nil))
}
