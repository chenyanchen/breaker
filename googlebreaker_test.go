package breaker

import (
	"errors"
	"math/rand"
	"testing"
)

var errTest = errors.New("test error")

func Test_googleBreaker_Do(t *testing.T) {
	type args struct {
		f func() error
	}
	tests := []struct {
		name            string
		breakerCreateFn func() *googleBreaker
		args            args
		wantErr         error
	}{
		{
			name:            "inner error",
			breakerCreateFn: func() *googleBreaker { return NewGoogleBreaker() },
			args:            args{func() error { return errTest }},
			wantErr:         errTest,
		}, {
			name: "0% drop ratio",
			breakerCreateFn: func() *googleBreaker {
				breaker := NewGoogleBreaker()
				for i := 0; i < 100; i++ {
					_ = breaker.Do(func() error { return nil })
				}
				return breaker
			},
			args:    args{f: func() error { return nil }},
			wantErr: nil,
		}, {
			// This case is not 100% accurate, but it is enough to prove that the drop ratio is close to 99%.
			name: "close 99% drop ratio",
			breakerCreateFn: func() *googleBreaker {
				breaker := NewGoogleBreaker(WithK(0.5))
				for i := 0; i < 100; i++ {
					_ = breaker.Do(func() error { return errTest })
				}
				return breaker
			},
			args:    args{f: func() error { return nil }},
			wantErr: ErrServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.breakerCreateFn()
			if err := b.Do(tt.args.f); !errors.Is(err, tt.wantErr) {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkGoogleBreaker_Do(b *testing.B) {
	breaker := NewGoogleBreaker()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = breaker.Do(func() error {
				if rand.Float64() > 0.5 {
					return errTest
				}
				return nil
			})
		}
	})
}
