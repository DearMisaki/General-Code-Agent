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
```

## Golden Eval

```bash
go run ./cmd/mew-eval --mode context --cases testdata/evals/context --out /tmp/context-eval.jsonl
go run ./cmd/mew-eval --mode teammate --cases testdata/evals/teammate --out /tmp/teammate-eval.jsonl
```
