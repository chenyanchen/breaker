package rand

import "testing"

func BenchmarkLockedSource_Int63(b *testing.B) {
	source := NewLockSource()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// lockedSource is thread-safe, should not panic.
			_ = source.Int63()
		}
	})
}
