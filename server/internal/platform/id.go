package platform

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
)

// ID is a server-generated UUIDv7 (RFC 9562, doc 06): time-ordered, so new
// rows land at the end of their indexes.
type ID [16]byte

// String returns the canonical lowercase 8-4-4-4-12 form.
func (id ID) String() string {
	var b [36]byte
	hex.Encode(b[0:8], id[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], id[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], id[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], id[8:10])
	b[23] = '-'
	hex.Encode(b[24:], id[10:])
	return string(b[:])
}

// IDGenerator makes UUIDv7s from its clock and crypto/rand. It is safe for
// concurrent use, and its IDs strictly increase.
type IDGenerator struct {
	clock Clock
	mu    sync.Mutex
	last  ID
}

// NewIDGenerator returns a generator stamping IDs with clock's time.
func NewIDGenerator(clock Clock) *IDGenerator { return &IDGenerator{clock: clock} }

// New returns a fresh UUIDv7: 48 bits of Unix milliseconds, then 74 random
// bits around the version and variant fields. When the clock hasn't moved
// past the previous ID's millisecond, the previous ID plus one is returned
// instead (RFC 9562 §6.2, method 3), so IDs always increase.
func (g *IDGenerator) New() ID {
	var id ID
	// crypto/rand.Read never returns an error (Go 1.24+).
	_, _ = rand.Read(id[6:])

	var ms [8]byte
	binary.BigEndian.PutUint64(ms[:], uint64(g.clock.Now().UnixMilli()))
	copy(id[0:6], ms[2:8])

	id[6] = id[6]&0x0f | 0x70 // version 7
	id[8] = id[8]&0x3f | 0x80 // variant 10

	g.mu.Lock()
	defer g.mu.Unlock()
	if bytes.Compare(id[:], g.last[:]) <= 0 {
		id = g.last.next()
	}
	g.last = id
	return id
}

// next is the following UUIDv7 in order: the 74 random bits plus one,
// carrying into the timestamp only on overflow, version and variant kept.
func (id ID) next() ID {
	// Random bits: id[6] low nibble, id[7], id[8] low 6 bits, id[9..15].
	for i := 15; i >= 9; i-- {
		id[i]++
		if id[i] != 0 {
			return id
		}
	}
	if low := (id[8] + 1) & 0x3f; low != 0 {
		id[8] = id[8]&0xc0 | low
		return id
	}
	id[8] &= 0xc0
	id[7]++
	if id[7] != 0 {
		return id
	}
	if low := (id[6] + 1) & 0x0f; low != 0 {
		id[6] = id[6]&0xf0 | low
		return id
	}
	id[6] &= 0xf0
	// 2^74 IDs in one millisecond: move to the next millisecond.
	for i := 5; i >= 0; i-- {
		id[i]++
		if id[i] != 0 {
			break
		}
	}
	return id
}

// ParseID reads a UUID in the canonical 8-4-4-4-12 hex form, either case.
func ParseID(s string) (ID, error) {
	var id ID
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return ID{}, fmt.Errorf("platform: %q is not a UUID", s)
	}
	compact := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:]
	if _, err := hex.Decode(id[:], []byte(compact)); err != nil {
		return ID{}, fmt.Errorf("platform: %q is not a UUID", s)
	}
	return id, nil
}
