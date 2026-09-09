# Task 9 Report: Add Benchmark Command Documentation

## What Was Implemented

- Created `docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md` with copy-paste commands for:
  - Context benchmarks.
  - Teammate benchmarks.
  - CPU and memory profiles.
  - Golden context and teammate evaluations through `cmd/mew-eval`.
- Appended the requested `## 6. 执行命令索引` section to `docs/superpowers/specs/2026-09-09-context-teammate-performance-evaluation.md`, linking to the command document.

## Verification Output

Ran the exact brief command:

```text
docs/superpowers/specs/2026-09-09-context-teammate-performance-evaluation.md:69:- `BenchmarkGatewayPrepareTurn`
docs/superpowers/specs/2026-09-09-context-teammate-performance-evaluation.md:72:- `BenchmarkGatewayPrepareTurnWithLargeConversation`
docs/superpowers/specs/2026-09-09-context-teammate-performance-evaluation.md:73:- `BenchmarkGatewayPrepareTurnWithManyDeferredTools`
docs/superpowers/specs/2026-09-09-context-teammate-performance-evaluation.md:581:## 6. 执行命令索引
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:6:go test ./internal/contextmgr -run '^$' -bench 'BenchmarkGatewayPrepareTurn' -benchmem -count=10
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:18:go test ./internal/contextmgr -run '^$' -bench BenchmarkGatewayPrepareTurn/messages_1000 -benchmem -cpuprofile /tmp/context-cpu.out -memprofile /tmp/context-mem.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:25:go run ./cmd/mew-eval --mode context --cases testdata/evals/context --out /tmp/context-eval.jsonl
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:26:go run ./cmd/mew-eval --mode teammate --cases testdata/evals/teammate --out /tmp/teammate-eval.jsonl
```

Also ran `git show --check --stat --oneline HEAD`; it completed without whitespace errors.

## Files Changed

- `docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md`
- `docs/superpowers/specs/2026-09-09-context-teammate-performance-evaluation.md`

## Self-Review Findings

- All command text matches the task brief verbatim.
- The spec link and required heading are present.
- No TUI-related code was read or modified.
- No unrelated worktree changes were staged or modified.
- No whitespace errors were reported by `git show --check`.

## Concerns

- The repository ignores `docs/`, and the original spec was untracked in this checkout. The requested commit therefore adds the full original spec file plus the new command file, resulting in 610 insertions. Only the two requested documentation paths were force-staged.
- Existing unrelated worktree changes remain present and were not included in the commit.

## Commit

- `427e012 docs: add context teammate performance commands`

## Fix Round 1

### Findings Addressed

- Added `-blockprofile` and `go tool pprof` commands for block profiles.
- Added `-mutexprofile` and `go tool pprof` commands for mutex profiles.
- Added `-trace` and `go tool trace` commands.
- Added a `-race` command covering the context, teams, and orchestration packages.
- Added `go tool pprof` inspection for the generated memory profile.

### Verification Output

Ran:

```text
Command: `rg -n -- "-blockprofile|-mutexprofile|-trace|-race|go tool pprof|go tool trace" docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md`

docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:19:go tool pprof -http=:0 /tmp/context-cpu.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:20:go tool pprof -http=:0 /tmp/context-mem.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:26:go test ./internal/contextmgr -run '^$' -bench BenchmarkGatewayPrepareTurn/messages_1000 -benchmem -blockprofile /tmp/context-block.out -mutexprofile /tmp/context-mutex.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:27:go tool pprof -http=:0 /tmp/context-block.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:28:go tool pprof -http=:0 /tmp/context-mutex.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:34:go test ./internal/contextmgr -run '^$' -bench BenchmarkGatewayPrepareTurn/messages_1000 -benchmem -trace /tmp/context-trace.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:35:go tool trace /tmp/context-trace.out
docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md:41:go test -race ./internal/contextmgr ./internal/teams ./internal/orchestration
```

`git diff --check` completed without whitespace errors.

### Files Changed

- `docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md`
- `.superpowers/sdd/2026-09-09-context-teammate-performance-evaluation/task-9-report.md`

### Self-Review Findings

- All five review findings are addressed with copy-paste commands.
- The original spec link section was not changed.
- No TUI-related code or unrelated files were modified.
