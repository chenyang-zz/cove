#!/usr/bin/env bash
set -euo pipefail

required_variables=(
  IOS_SIMULATOR_UDID
  MAESTRO_EXPO_DEV_CLIENT_URL
  MAESTRO_E2E_API_URL
  MAESTRO_E2E_USERNAME
  MAESTRO_E2E_EMAIL
  MAESTRO_E2E_PASSWORD
  MAESTRO_E2E_KNOWLEDGE_NAME
  MAESTRO_E2E_KNOWLEDGE_DESCRIPTION
)

for variable_name in "${required_variables[@]}"; do
  if [[ -z "${!variable_name:-}" ]]; then
    echo "Missing required environment variable: ${variable_name}" >&2
    exit 2
  fi
done

for command_name in maestro node xcrun; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Required command not found: ${command_name}" >&2
    exit 2
  fi
done

mobile_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
workspace_root="$(cd "${mobile_dir}/../../.." && pwd)"
run_id="${E2E_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)-knowledge-create-detail}"

if [[ ! "${run_id}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "E2E_RUN_ID may contain only letters, numbers, dots, underscores, and hyphens." >&2
  exit 2
fi

artifact_dir="${workspace_root}/output/ios-simulator/runs/${run_id}"
recording_path="${artifact_dir}/evidence/knowledge-create-detail.mp4"
recording_pid=""
mkdir -p "${artifact_dir}/evidence" "${artifact_dir}/maestro-debug"

cleanup() {
  local exit_status=$?
  trap - EXIT
  if [[ -n "${recording_pid}" ]] && kill -0 "${recording_pid}" 2>/dev/null; then
    kill -INT "${recording_pid}" 2>/dev/null || true
    wait "${recording_pid}" 2>/dev/null || true
  fi
  if ! node "${mobile_dir}/e2e/maestro/sanitize-artifacts.mjs" "${artifact_dir}"; then
    echo "Failed to sanitize Maestro artifacts: ${artifact_dir}" >&2
    exit_status=1
  fi
  exit "${exit_status}"
}

trap cleanup EXIT

export MAESTRO_CLI_NO_ANALYTICS="${MAESTRO_CLI_NO_ANALYTICS:-true}"
export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED="${MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED:-true}"

node "${mobile_dir}/e2e/maestro/setup-knowledge-fixture.mjs"

xcrun simctl io "${IOS_SIMULATOR_UDID}" recordVideo \
  --codec=h264 \
  --force \
  "${recording_path}" \
  >"${artifact_dir}/recording.log" 2>&1 &
recording_pid=$!

if ! kill -0 "${recording_pid}" 2>/dev/null; then
  wait "${recording_pid}" || true
  echo "Failed to start Simulator recording. See ${artifact_dir}/recording.log." >&2
  exit 1
fi

echo "Running Cove knowledge create/detail E2E on Simulator ${IOS_SIMULATOR_UDID}"
echo "Artifacts: ${artifact_dir}"

maestro test \
  --udid "${IOS_SIMULATOR_UDID}" \
  --config "${mobile_dir}/e2e/maestro/config.yaml" \
  --test-output-dir "${artifact_dir}/evidence" \
  --debug-output "${artifact_dir}/maestro-debug" \
  --format JUNIT \
  --output "${artifact_dir}/maestro-junit.xml" \
  "${mobile_dir}/e2e/maestro/flows/knowledge-create-detail.yaml"
