# Context Teammate Performance Commands

## Context Benchmarks

```bash
go test ./internal/contextmgr -run '^$' -bench 'BenchmarkGatewayPrepareTurn' -benchmem -count=10
```

## Teammate Benchmarks

```bash
go test ./internal/teams ./internal/orchestration -run '^$' -bench 'Benchmark(FileMailBox|TaskBoard)' -benchmem -count=10
```

## Profiles

```bash
go test ./internal/contextmgr -run '^$' -bench BenchmarkGatewayPrepareTurn/messages_1000 -benchmem -cpuprofile /tmp/context-cpu.out -memprofile /tmp/context-mem.out
go tool pprof -http=:0 /tmp/context-cpu.out
go tool pprof -http=:0 /tmp/context-mem.out
```

## Block and Mutex Profiles

```bash
go test ./internal/contextmgr -run '^$' -bench BenchmarkGatewayPrepareTurn/messages_1000 -benchmem -blockprofile /tmp/context-block.out -mutexprofile /tmp/context-mutex.out
go tool pprof -http=:0 /tmp/context-block.out
go tool pprof -http=:0 /tmp/context-mutex.out
```

## Runtime Trace

```bash
go test ./internal/contextmgr -run '^$' -bench BenchmarkGatewayPrepareTurn/messages_1000 -benchmem -trace /tmp/context-trace.out
go tool trace /tmp/context-trace.out
```

## Race Detection

```bash
go test -race ./internal/contextmgr ./internal/teams ./internal/orchestration
```

## Golden Eval

```bash
go run ./cmd/mew-eval --mode context --cases testdata/evals/context --out /tmp/context-eval.jsonl
go run ./cmd/mew-eval --mode teammate --cases testdata/evals/teammate --out /tmp/teammate-eval.jsonl
```
