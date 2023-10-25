package rand

import (
	"math/rand"
	"sync"
)

// Why not use rand.Int63() and others functions from rand.*?
// Cause the rand.globalRand use a global lock in rand.lockedSource,
// that may have a performance impact in a big project.

// Why not use rand.NewSource()?
// Cause math.NewSource are not safe for concurrent use by multiple goroutines.
// See: https://github.com/golang/go/blob/master/src/math/rand/rand.go#L47

// lockedSource is a rand.Source implementation that is safe for concurrent
// use by multiple goroutines.
// The code is partial copied from rand.lockedSource.
type lockedSource struct {
	lk sync.Mutex
	s  rand.Source
}

func NewLockSource() *lockedSource {
	return &lockedSource{s: rand.NewSource(rand.Int63())}
}

func (r *lockedSource) Int63() (n int64) {
	r.lk.Lock()
	n = r.s.Int63()
	r.lk.Unlock()
	return n
}

func (r *lockedSource) Seed(seed int64) {
	r.lk.Lock()
	r.s.Seed(seed)
	r.lk.Unlock()
}
