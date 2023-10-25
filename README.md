# What is this?

Circuit Breaker in Go.

# Why use it?

A grace way to Handling Overload in client-side.

# How does it work?

There are only one implementation of Circuit Breaker, it is from [Google SRE](https://sre.google/sre-book/handling-overload).

# How to use it?

The abstract of Breaker interface is clear, it only cares about:

- the dependency is available or not

Not care about:

- specific errors
- fallback strategies
- telemetry

There are some examples to show how to use it:

- Use Circuit Breaker to protect your service (e.g. [example/simple/breaker.go](example/simple/breaker.go))
- Handle specific errors (e.g. [example/acceptableerror/breaker.go](example/acceptableerror/breaker.go))
- Add fallback strategies (e.g. [example/fallback/breaker.go](example/fallback/breaker.go))
- Add telemetry middleware (e.g. [example/telemetry/breaker.go](example/telemetry/breaker.go))

# Benchmark

```bash
❯ go test -bench=. -benchmem
goos: darwin
goarch: arm64
pkg: github.com/chenyanchen/breaker
BenchmarkGoogleBreaker_Do-8      5794507               249.1 ns/op             0 B/op          0 allocs/op
PASS
ok      github.com/chenyanchen/breaker  1.658s
```
