#!/usr/bin/env bash
set -euo pipefail

case "$(go env GOVERSION)" in
  go1.27.*) ;;
  *) echo "Go 1.27 is required" >&2; exit 1 ;;
esac
test "$(git ls-files 'meta-ontology-go/**' | wc -l | tr -d ' ')" = "0"
test "$(git grep -l 'ontology_reference.*meta-ontology-go' -- '*.json' | wc -l | tr -d ' ')" -ge 1
test "$(git grep -l 'repository_writes.*0' -- 'fixtures/cases/*/*.json' | wc -l | tr -d ' ')" -ge 1
test -z "$(git diff --check)"
test "$(git grep -l 'source_text_merged.*false' -- README.md | wc -l | tr -d ' ')" = "0"
echo "CI policy: Go 1.27, external read-only ontology, zero input writes"
