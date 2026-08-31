#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

run_case() {
  local case_name="$1"
  local authority_file="$2"
  local expected_state="$3"
  local output_dir="${work_dir}/${case_name}"
  go run ./cmd/gooo-semantic-merge-advisor plan \
    --left "fixtures/cases/${case_name}/left.json" \
    --right "fixtures/cases/${case_name}/right.json" \
    --authority "fixtures/cases/${authority_file}" \
    --source examples/semantic-merge-authority/main.gooo \
    --denominator contracts/semantic-merge-advisor-denominator-v1.json \
    --output "${output_dir}"

  jq -e --arg expected "${expected_state}" \
    '.state == $expected and .improvement == "UNKNOWN" and .source_text_merged == false and .input_repository_writes == 0 and .metrics.repository_writes == 0 and .metrics.local_tests_run == 0 and .metrics.artifact_files == 3 and .metrics.artifact_bytes > 0 and (all(.metrics.phase_wall_ms[]; . >= 0))' \
    "${output_dir}/merge-proposal.json" >/dev/null
  jq -e --arg expected "${expected_state}" \
    '.state == $expected and .improvement == "UNKNOWN" and .source_text_merged == false and .repository_writes == 0 and .input_repository_writes == 0 and .metrics.artifact_files == 3 and .metrics.artifact_bytes > 0' \
    "${output_dir}/authority-receipt.json" >/dev/null
  jq -e --arg expected "${expected_state}" \
    '.state == $expected and .input_repository_writes == 0 and .metrics.repository_writes == 0 and .metrics.artifact_files == 3' \
    "${output_dir}/counterexample-report.json" >/dev/null
}

cd "${root_dir}"
run_case normal authority.json CLOSED
run_case unknown authority.json UNKNOWN
run_case refuted authority.json REFUTED

jq -e 'all(.cardinality[]; .exactly_one == true)' "${work_dir}/normal/merge-proposal.json" >/dev/null
jq -e 'all(.unknowns[]; ((.unknown | keys | sort) == ["blocked_by", "next_operation", "reason", "stage", "step", "unknown_class"]))' "${work_dir}/unknown/counterexample-report.json" >/dev/null
jq -e '([.refutations[].reasons[]] | unique) as $reasons | (["stale_api", "same_authority_twice", "new_and_previous_file_double_ownership", "multiple_authorities"] | all(. as $reason | $reasons | index($reason) != null))' "${work_dir}/refuted/counterexample-report.json" >/dev/null

if go run ./cmd/gooo-semantic-merge-advisor plan \
  --left fixtures/cases/malformed/left.json \
  --right fixtures/cases/malformed/right.json \
  --authority fixtures/cases/authority.json \
  --source examples/semantic-merge-authority/main.gooo \
  --denominator contracts/semantic-merge-advisor-denominator-v1.json \
  --output "${work_dir}/malformed" >/dev/null 2>&1; then
  echo "malformed input unexpectedly succeeded" >&2
  exit 1
fi

if go run ./cmd/gooo-semantic-merge-advisor plan \
  --left fixtures/cases/fixed-point/left.json \
  --right fixtures/cases/fixed-point/right.json \
  --authority fixtures/cases/fixed-point/authority.json \
  --source examples/semantic-merge-authority/main.gooo \
  --denominator contracts/semantic-merge-advisor-denominator-v1.json \
  --output "${work_dir}/fixed-point" >/dev/null 2>&1; then
  echo "FIXED_POINT input unexpectedly succeeded" >&2
  exit 1
fi

echo "semantic conformance: normal, UNKNOWN, REFUTED, malformed, FIXED_POINT"
