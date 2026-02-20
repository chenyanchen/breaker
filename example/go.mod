module github.com/chenyanchen/breaker/example

go 1.22

require (
	github.com/chenyanchen/breaker v0.0.1
	go.opentelemetry.io/otel/metric v1.19.0
)

require go.opentelemetry.io/otel v1.19.0 // indirect

replace github.com/chenyanchen/breaker => ../
