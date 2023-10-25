package breaker

import "errors"

type Breaker interface {
	Do(func() error) error
}

var ErrServiceUnavailable = errors.New("circuit breaker is open")
