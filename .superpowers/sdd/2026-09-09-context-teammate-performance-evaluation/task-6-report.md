# Task 6 Report: Teammate Golden Eval Scorer

## What I implemented

- Added the requested teammate evaluation data types and JSON tags.
- Added deterministic `ScoreTeammateEval` scoring for:
  - tool precision, recall, and F1;
  - message delivery accuracy;
  - collaboration completion from task-transition matches;
  - repeated-message loop detection.
- Tool calls use one-to-one matching by tool name and expected-argument subset.
- Messages use one-to-one matching for non-empty expected `from`, `to`, and `task_id` fields.
- Task transitions use exact one-to-one matching on `task_id`, `from_status`, and `to_status`.
- No LLM, mem0, network, or TUI code was used or changed.

## Test results

- `go test ./internal/evals -run TestScoreTeammateEval -count=1` — PASS.
- `env GOCACHE=/private/tmp/mewcode-task6-gocache go test ./internal/evals -count=1` — PASS.
- An initial full-package run without the explicit cache failed before test execution because the environment denied access to the default Go build-cache path; the writable-cache rerun passed.

## TDD evidence

1. Added `TestScoreTeammateEvalComputesToolF1AndDeliveryAccuracy` before production code.
2. Ran the focused test and observed the expected RED build failure: the scorer function and types were undefined.
3. Implemented the minimal scorer required by the brief.
4. Reran the focused test and observed PASS, then ran the full `internal/evals` package successfully.

## Files changed

- `internal/evals/teammate_scorer.go`
- `internal/evals/teammate_scorer_test.go`
- `.superpowers/sdd/2026-09-09-context-teammate-performance-evaluation/task-6-report.md`

## Self-review findings

- The implementation is deterministic and package-local.
- Matching is one-to-one, so duplicate observations cannot inflate recall or delivery/completion scores.
- Empty denominators return zero through the existing package `ratio` helper.
- Loop detection triggers only after the same `(from, to, text)` tuple appears more than twice.
- No unrelated dirty files were staged or modified.

## Concerns

- `MaxAllowedMessages` is represented as required but is not used in scoring because the brief specifies no behavior for it.

## Fix Round 1

### Findings addressed

- Fixed expected tool-argument matching so an expected key must exist in the observed argument map, including when its expected value is an empty string.
- Added focused coverage for:
  - missing observed keys with expected empty-string values;
  - wildcard behavior for empty expected message fields;
  - partial collaboration completion from task transitions;
  - the loop threshold of more than two identical messages;
  - one-to-one duplicate tool matching.

### Command output

- `env GOCACHE=/private/tmp/mewcode-task6-fix-gocache go test ./internal/evals -run TestScoreTeammateEval -count=1`

  `ok   mewcode/internal/evals  0.305s`

- `env GOCACHE=/private/tmp/mewcode-task6-fix-gocache go test ./internal/evals -count=1`

  `ok   mewcode/internal/evals  0.193s`

### TDD evidence

The new missing-key regression test failed before the production change with a false `ToolPrecision: 1` and `ToolRecall: 1`. After the two-value lookup fix, the focused suite passed. The remaining focused tests passed alongside it and lock down the requested matching rules.
