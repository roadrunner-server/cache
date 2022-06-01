package hasher

import (
	"hash"
	"hash/fnv"
	"sync"
)

type Hasher struct {
	pool sync.Pool
}

func NewHasher() *Hasher {
	return &Hasher{
		pool: sync.Pool{
			New: func() interface{} {
				return fnv.New64a()
			},
		},
	}
}

func (hs *Hasher) GetHash() hash.Hash64 {
	return hs.pool.Get().(hash.Hash64)
}

func (hs *Hasher) PutHash(h hash.Hash64) {
	h.Reset()
	hs.pool.Put(h)
}
