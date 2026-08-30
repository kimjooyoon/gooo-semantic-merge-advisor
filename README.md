# Gooo Semantic Merge Advisor

`gooo-semantic-merge-advisor` creates a semantic authority-union plan from two
immutable Git tree manifests and a Gooo authority declaration. It never merges
source text. Instead, it proves whether each symbol, claim, producer, and
evaluator has exactly one owner, preserves unobservable bindings as explicit
six-field `UNKNOWN` records, and rejects stale or multiply-owned facts as
`REFUTED`.

The top-level precedence is strict:

```text
REFUTED > UNKNOWN > CLOSED
```

The generated output directory contains three machine-readable artifacts:

- `merge-proposal.json`: proposed semantic actions and cardinality evidence;
- `authority-receipt.json`: immutable input digests, phase timings, resource
  observations, and the zero-write boundary;
- `counterexample-report.json`: every refutation and unknown witness.

## Usage

```text
go run ./cmd/gooo-semantic-merge-advisor plan \
  --left fixtures/cases/normal/left.json \
  --right fixtures/cases/normal/right.json \
  --authority fixtures/cases/normal/authority.json \
  --source examples/semantic-merge-authority/main.gooo \
  --denominator contracts/semantic-merge-advisor-denominator-v1.json \
  --output ./out
```

Inputs are read-only. Only `--output` is written, and `repository_writes` is
always recorded as the integer `0`. The source text is used only for its digest
and source-line binding; no source text is emitted into a proposal.

The reference to `meta-ontology-go` is deliberately external and read-only.
This repository does not vendor, import, or modify that repository.

## CI-only verification

Go 1.27 format, vet, tests, race tests, and all conformance cases run in GitHub
Actions. They are intentionally not run in the authoring worktree; the receipt
records `local_tests_run: 0`. CI covers normal, `UNKNOWN`, `REFUTED`, malformed,
and `FIXED_POINT` inputs, including stale helper and duplicate
`candidateDefinition` fixtures.
