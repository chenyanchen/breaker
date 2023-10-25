package breaker

import (
	"math/rand"
	"time"

	"github.com/chenyanchen/breaker/internal/rollingwindow"
)

const (
	defaultK        = 1.5
	defaultSize     = 20
	defaultInterval = time.Millisecond * 500
)

type googleBreaker struct {
	rand *rand.Rand

	k float64

	stat *rollingwindow.RollingWindow
}

type Option func(*googleBreaker)

func WithK(k float64) Option {
	return func(b *googleBreaker) { b.k = k }
}

func WithWindow(size int, interval time.Duration) Option {
	return func(b *googleBreaker) {
		b.stat = rollingwindow.NewRollingWindow(size, interval)
	}
}

func NewGoogleBreaker(opts ...Option) *googleBreaker {
	b := &googleBreaker{
		k:    defaultK,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
		stat: rollingwindow.NewRollingWindow(defaultSize, defaultInterval),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

func (b *googleBreaker) Do(f func() error) error {
	if err := b.accept(); err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			b.markFailure()
			panic(v)
		}
	}()

	err := f()
	if err != nil {
		b.markFailure()
	} else {
		b.markSuccess()
	}

	return err
}

func (b *googleBreaker) accept() error {
	accepts, requests := b.history()

	// https://sre.google/sre-book/handling-overload/#eq2101
	dropRatio := (requests - b.k*accepts) / (requests + 1)
	if dropRatio <= 0 {
		return nil
	}

	if b.rand.Float64() < dropRatio {
		return ErrServiceUnavailable
	}

	return nil
}

func (b *googleBreaker) markSuccess() { b.stat.Add(1) }
func (b *googleBreaker) markFailure() { b.stat.Add(0) }

func (b *googleBreaker) history() (accepts, requests float64) {
	b.stat.Reduce(func(b *rollingwindow.Bucket) {
		accepts += b.Value
		requests += b.Count
	})
	return
}
