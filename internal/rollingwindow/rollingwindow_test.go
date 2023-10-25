package rollingwindow

import (
	"testing"
	"time"
)

const span = time.Millisecond * 10

func TestRollingWindow_Reduce(t *testing.T) {
	tests := []struct {
		name           string
		windowCreateFn func() *RollingWindow
		wantCount      float64
		wantSum        float64
	}{
		{
			name: "all buckets are valid",
			windowCreateFn: func() *RollingWindow {
				rollingWindow := NewRollingWindow(2, span)
				rollingWindow.Add(1 << 0)
				rollingWindow.Add(1 << 1)
				return rollingWindow
			},
			wantCount: 2,
			wantSum:   3,
		}, {
			name: "all buckets are invalid",
			windowCreateFn: func() *RollingWindow {
				rollingWindow := NewRollingWindow(2, span)
				rollingWindow.Add(1 << 0)
				rollingWindow.Add(1 << 1)
				time.Sleep(span)
				return rollingWindow
			},
			wantCount: 0,
			wantSum:   0,
		}, {
			name: "case 3",
			windowCreateFn: func() *RollingWindow {
				rollingWindow := NewRollingWindow(2, span)
				rollingWindow.Add(1 << 0)
				time.Sleep(span)
				rollingWindow.Add(1 << 1)
				return rollingWindow
			},
			wantCount: 1,
			wantSum:   2,
		}, {
			name: "expire all buckets and add new buckets",
			windowCreateFn: func() *RollingWindow {
				rollingWindow := NewRollingWindow(2, span)
				rollingWindow.Add(1 << 0)
				time.Sleep(span)
				rollingWindow.Add(1 << 1)
				time.Sleep(span * 3)
				rollingWindow.Add(1 << 2)
				return rollingWindow
			},
			wantCount: 1,
			wantSum:   4,
		}, {
			name: "reduce all expired buckets",
			windowCreateFn: func() *RollingWindow {
				rollingWindow := NewRollingWindow(2, span)
				rollingWindow.Add(1 << 0)
				time.Sleep(span)
				rollingWindow.Add(1 << 1)
				time.Sleep(span * 3)
				return rollingWindow
			},
			wantCount: 0,
			wantSum:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rollingWindow := tt.windowCreateFn()
			var sum, count float64
			rollingWindow.Reduce(func(bucket *Bucket) {
				sum += bucket.Value
				count += bucket.Count
			})
			if count != tt.wantCount {
				t.Errorf("Reduce() count = %v, wantCount %v", count, tt.wantCount)
				return
			}
			if sum != tt.wantSum {
				t.Errorf("Reduce() sum = %v, wantSum %v", sum, tt.wantSum)
			}
		})
	}
}
