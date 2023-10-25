package rollingwindow

import (
	"sync"
	"time"
)

// RollingWindow defines a thread-safe rolling window to calculate
// the events in buckets with time interval.
type RollingWindow struct {
	lock sync.RWMutex

	// window size
	size int

	// bucket time interval
	interval time.Duration

	// current bucket offset
	offset int

	buckets []*Bucket

	// last update time
	lastTime time.Time
}

// NewRollingWindow returns a RollingWindow that with size buckets and time interval.
func NewRollingWindow(size int, interval time.Duration) *RollingWindow {
	if size < 1 {
		panic("size must be greater than 0")
	}

	w := &RollingWindow{
		size:     size,
		interval: interval,
		buckets:  make([]*Bucket, size),
		lastTime: time.Now(),
	}

	for i := range w.buckets {
		w.buckets[i] = &Bucket{}
	}

	return w
}

func (w *RollingWindow) Add(v float64) {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.updateOffset()

	// Add value to current Bucket.
	w.buckets[w.offset].Value += v
	w.buckets[w.offset].Count++
}

// updateOffset updates the offset of current bucket.
func (w *RollingWindow) updateOffset() {
	// Calculate window span.
	span := w.span()
	if span <= 0 {
		return
	}

	if span > w.size {
		span = w.size
	}

	// Reset expired buckets.
	for i := 0; i < span; i++ {
		w.buckets[(w.offset+i)%w.size].Reset()
	}

	// Move offset.
	w.offset = (w.offset + span) % w.size

	// Update last update time.
	w.lastTime = time.Now().Truncate(w.interval)
}

func (w *RollingWindow) Reduce(fn func(bucket *Bucket)) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	span := w.span()
	if span >= w.size {
		return
	}

	for i := 0; i < w.size-span; i++ {
		bucket := w.buckets[(w.offset+span+i)%w.size]
		fn(bucket)
	}
}

func (w *RollingWindow) span() int {
	return int(time.Since(w.lastTime) / w.interval)
}

type Bucket struct {
	Value float64
	Count float64
}

func (b *Bucket) Reset() {
	b.Value, b.Count = 0, 0
}
