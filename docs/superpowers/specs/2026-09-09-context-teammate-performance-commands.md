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

## Initial Local Baseline

- Machine: darwin/arm64, Apple M1.
- Go: go1.27.1 darwin/arm64.
- Context benchmark one-run summary: `BenchmarkGatewayPrepareTurn/messages_1000` 15.282473 ms/op, 1000623 B/op, 264 allocs/op.
- Teammate benchmark one-run summary: `BenchmarkFileMailBoxSendParallel` 1.045508 ms/op, 0 lock_errors/op, 718803 B/op, 767 allocs/op; `BenchmarkTaskBoardClaimContention/contenders_100` 222.332083 ms/op, 369546 B/op, 2988 allocs/op.
- Golden eval smoke: context and teammate JSONL outputs were non-empty and contained score records with expected metric fields.
