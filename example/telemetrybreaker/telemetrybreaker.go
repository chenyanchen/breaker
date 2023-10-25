package telemetrybreaker

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/metric"

	"github.com/chenyanchen/breaker"
)

type telemetryBreaker struct {
	successCounter metric.Int64Counter
	dropCounter    metric.Int64Counter
	failureCounter metric.Int64Counter

	breaker breaker.Breaker
}

func NewTelemetryBreaker(breaker breaker.Breaker, meter metric.Meter) (breaker.Breaker, error) {
	successCounter, err := meter.Int64Counter("breaker.success")
	if err != nil {
		return nil, fmt.Errorf("failed to create success counter: %w", err)
	}
	dropCounter, err := meter.Int64Counter("breaker.drop")
	if err != nil {
		return nil, fmt.Errorf("failed to create drop counter: %w", err)
	}
	failureCounter, err := meter.Int64Counter("breaker.failure")
	if err != nil {
		return nil, fmt.Errorf("failed to create failure counter: %w", err)
	}

	return &telemetryBreaker{
		successCounter: successCounter,
		dropCounter:    dropCounter,
		failureCounter: failureCounter,
		breaker:        breaker,
	}, nil
}

func (b *telemetryBreaker) Do(f func() error) error {
	err := b.breaker.Do(f)

	ctx := context.Background()

	if err == nil {
		b.successCounter.Add(ctx, 1)
		return nil
	}

	if errors.Is(err, breaker.ErrServiceUnavailable) {
		b.dropCounter.Add(ctx, 1)
		return err
	}

	b.failureCounter.Add(ctx, 1)
	return err
}
