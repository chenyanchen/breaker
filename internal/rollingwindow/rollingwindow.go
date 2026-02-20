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

	now func() time.Time
}

// NewRollingWindow returns a RollingWindow that with size buckets and time interval.
func NewRollingWindow(size int, interval time.Duration) *RollingWindow {
	return newRollingWindow(size, interval, time.Now)
}

func newRollingWindow(size int, interval time.Duration, now func() time.Time) *RollingWindow {
	if size < 1 {
		panic("size must be greater than 0")
	}

	if now == nil {
		now = time.Now
	}

	w := &RollingWindow{
		size:     size,
		interval: interval,
		buckets:  make([]*Bucket, size),
		lastTime: now(),
		now:      now,
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
	w.lastTime = w.now().Truncate(w.interval)
}

func (w *RollingWindow) Reduce(fn func(bucket *Bucket)) {
	w.lock.RLock()
	span := w.span()
	if span >= w.size {
		w.lock.RUnlock()
		return
	}

	snapshot := make([]Bucket, 0, w.size-span)
	for i := 0; i < w.size-span; i++ {
		bucket := w.buckets[(w.offset+span+i)%w.size]
		snapshot = append(snapshot, *bucket)
	}
	w.lock.RUnlock()

	for i := range snapshot {
		fn(&snapshot[i])
	}
}

func (w *RollingWindow) span() int {
	return int(w.now().Sub(w.lastTime) / w.interval)
}

type Bucket struct {
	Value float64
	Count float64
}

func (b *Bucket) Reset() {
	b.Value, b.Count = 0, 0
}
