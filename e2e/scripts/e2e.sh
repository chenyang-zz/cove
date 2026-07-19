#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
workspace_root="$(cd -- "${script_dir}/../.." && pwd)"
compose_file="${workspace_root}/e2e/compose.yml"
config_file="${workspace_root}/e2e/config.yml"
server_dir="${workspace_root}/packages/server"
frontend_dir="${workspace_root}/packages/app/frontend"
artifact_root="${workspace_root}/output/playwright"
orbstack_context="${E2E_ORBSTACK_CONTEXT:-orbstack}"
api_pid=""
web_pid=""
llm_pid=""
worker_pid=""
compose_log=""

command_name="${1:-}"
if [[ -z "${command_name}" ]]; then
  echo "usage: $0 <up|app-backend|server-db-smoke|smoke|logs|down>" >&2
  exit 2
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

require_command curl
require_command docker
require_command orb
require_command go
require_command node
if [[ "${command_name}" == "smoke" ]]; then
  require_command pnpm
fi

free_port() {
  node -e 'const net=require("node:net");const s=net.createServer();s.listen(0,"127.0.0.1",()=>{process.stdout.write(String(s.address().port));s.close();});'
}

if [[ "${command_name}" == "smoke" || "${command_name}" == "server-db-smoke" || "${command_name}" == "app-backend" ]]; then
  e2e_run_id="${E2E_RUN_ID:-$(date +%Y%m%d%H%M%S)-$$}"
  e2e_project="${E2E_PROJECT:-cove-e2e-${e2e_run_id}}"
  export E2E_POSTGRES_PORT="${E2E_POSTGRES_PORT:-$(free_port)}"
  export E2E_REDIS_PORT="${E2E_REDIS_PORT:-$(free_port)}"
  export E2E_ELASTICSEARCH_PORT="${E2E_ELASTICSEARCH_PORT:-$(free_port)}"
  e2e_api_port="${E2E_API_PORT:-$(free_port)}"
  e2e_web_port="${E2E_WEB_PORT:-$(free_port)}"
  e2e_llm_port="${E2E_LLM_PORT:-$(free_port)}"
else
  e2e_run_id="${E2E_RUN_ID:-manual}"
  e2e_project="${E2E_PROJECT:-cove-e2e}"
  export E2E_POSTGRES_PORT="${E2E_POSTGRES_PORT:-55432}"
  export E2E_REDIS_PORT="${E2E_REDIS_PORT:-56379}"
  export E2E_ELASTICSEARCH_PORT="${E2E_ELASTICSEARCH_PORT:-59200}"
  e2e_api_port="${E2E_API_PORT:-58000}"
  e2e_web_port="${E2E_WEB_PORT:-55173}"
  e2e_llm_port="${E2E_LLM_PORT:-58001}"
fi

e2e_api_url="http://127.0.0.1:${e2e_api_port}"
e2e_web_url="http://127.0.0.1:${e2e_web_port}"
e2e_llm_url="http://127.0.0.1:${e2e_llm_port}/v1"
e2e_database_url="postgres://cove_e2e:cove_e2e@127.0.0.1:${E2E_POSTGRES_PORT}/cove_e2e?sslmode=disable"
run_artifact_dir="${artifact_root}/runs/${e2e_run_id}"
runtime_dir="${run_artifact_dir}/runtime"
log_dir="${run_artifact_dir}/logs"
storage_dir="${run_artifact_dir}/storage"

compose() {
  docker --context "${orbstack_context}" compose \
    --project-name "${e2e_project}" \
    --file "${compose_file}" \
    "$@"
}

ensure_orbstack() {
  local timeout_seconds="${E2E_ORBSTACK_START_TIMEOUT_SECONDS:-60}"
  local deadline=$((SECONDS + timeout_seconds))

  if docker --context "${orbstack_context}" info >/dev/null 2>&1; then
    return 0
  fi

  echo "Starting OrbStack..."
  if ! orb start >/dev/null 2>&1; then
    echo "Failed to start OrbStack with 'orb start'." >&2
    exit 1
  fi

  while ((SECONDS < deadline)); do
    if docker --context "${orbstack_context}" info >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "OrbStack did not become ready within ${timeout_seconds} seconds." >&2
  orb status >&2 || true
  exit 1
}

print_environment() {
  echo "E2E project: ${e2e_project}"
  echo "Container runtime: OrbStack (context: ${orbstack_context})"
  echo "Postgres: 127.0.0.1:${E2E_POSTGRES_PORT}"
  echo "Redis: 127.0.0.1:${E2E_REDIS_PORT}"
  echo "Elasticsearch: http://127.0.0.1:${E2E_ELASTICSEARCH_PORT}"
  echo "API: ${e2e_api_url}"
  if [[ "${command_name}" == "app-backend" || "${command_name}" == "server-db-smoke" ]]; then
    echo "Deterministic LLM: ${e2e_llm_url}"
  fi
  echo "Web: ${e2e_web_url}"
  echo "Artifacts: ${run_artifact_dir}"
}

server_env() {
  local app_host="127.0.0.1"
  if [[ "${command_name}" == "app-backend" ]]; then
    app_host="${E2E_APP_HOST:-0.0.0.0}"
  fi
  env \
    CONFIG_PATH="${config_file}" \
    APP_ENV=test \
    APP_HOST="${app_host}" \
    APP_PORT="${e2e_api_port}" \
    DATABASE_URL="${e2e_database_url}" \
    REDIS_ADDR="127.0.0.1:${E2E_REDIS_PORT}" \
    ES_HOST="http://127.0.0.1:${E2E_ELASTICSEARCH_PORT}" \
    JWT_SECRET=cove-e2e-jwt-secret \
    SECRET_KEY=0123456789abcdef0123456789abcdef \
    STORAGE_BACKEND=local \
    STORAGE_DIR="${storage_dir}" \
    RAG_CHUNK_INDEX="cove_e2e_${e2e_run_id}" \
    "$@"
}

wait_for_url() {
  local label="$1"
  local url="$2"
  local pid="$3"
  local log_file="$4"
  local timeout_seconds="${E2E_START_TIMEOUT_SECONDS:-120}"
  local deadline=$((SECONDS + timeout_seconds))

  while ((SECONDS < deadline)); do
    if curl --fail --silent --show-error --max-time 2 "${url}" >/dev/null 2>&1; then
      return 0
    fi
    if [[ -n "${pid}" ]] && ! kill -0 "${pid}" >/dev/null 2>&1; then
      echo "${label} exited before becoming ready. Last log lines:" >&2
      tail -n 80 "${log_file}" >&2 || true
      return 1
    fi
    sleep 0.5
  done

  echo "timed out waiting for ${label} at ${url}" >&2
  tail -n 80 "${log_file}" >&2 || true
  return 1
}

wait_for_log() {
  local label="$1"
  local pattern="$2"
  local pid="$3"
  local log_file="$4"
  local timeout_seconds="${E2E_START_TIMEOUT_SECONDS:-120}"
  local deadline=$((SECONDS + timeout_seconds))

  while ((SECONDS < deadline)); do
    if grep -q -- "${pattern}" "${log_file}" 2>/dev/null; then
      return 0
    fi
    if [[ -n "${pid}" ]] && ! kill -0 "${pid}" >/dev/null 2>&1; then
      echo "${label} exited before becoming ready. Last log lines:" >&2
      tail -n 80 "${log_file}" >&2 || true
      return 1
    fi
    sleep 0.5
  done

  echo "timed out waiting for ${label} log pattern: ${pattern}" >&2
  tail -n 80 "${log_file}" >&2 || true
  return 1
}

stop_pid() {
  local pid="$1"
  if [[ -z "${pid}" ]] || ! kill -0 "${pid}" >/dev/null 2>&1; then
    return 0
  fi
  pkill -TERM -P "${pid}" >/dev/null 2>&1 || true
  kill -TERM "${pid}" >/dev/null 2>&1 || true
  wait "${pid}" >/dev/null 2>&1 || true
}

start_dependencies() {
  ensure_orbstack
  compose up --detach --wait --wait-timeout "${E2E_START_TIMEOUT_SECONDS:-120}"
}

run_smoke() {
  api_pid=""
  web_pid=""
  llm_pid=""
  worker_pid=""
  local api_log="${log_dir}/api.log"
  local web_log="${log_dir}/web.log"
  local migration_log="${log_dir}/migration.log"
  local server_real_db_log="${log_dir}/server-real-db.log"
  local llm_log="${log_dir}/fake-openai.log"
  local worker_log="${log_dir}/worker.log"
  compose_log="${log_dir}/compose.log"

  mkdir -p "${runtime_dir}" "${log_dir}" "${storage_dir}"
  ln -sfn "${run_artifact_dir}" "${artifact_root}/latest"
  print_environment | tee "${run_artifact_dir}/environment.txt"

  cleanup() {
    local status=$?
    trap - EXIT
    stop_pid "${web_pid}"
    stop_pid "${api_pid}"
    stop_pid "${worker_pid}"
    stop_pid "${llm_pid}"
    compose logs --no-color >"${compose_log}" 2>&1 || true
    if [[ "${E2E_KEEP_ENV:-0}" != "1" ]]; then
      compose down --volumes --remove-orphans >/dev/null 2>&1 || true
    else
      echo "E2E_KEEP_ENV=1; dependency containers remain running as project ${e2e_project}."
    fi
    exit "${status}"
  }
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  start_dependencies

  (
    cd "${server_dir}"
    go build -o "${runtime_dir}/cove-migration" ./cmd/migration
    go build -o "${runtime_dir}/cove-api" ./cmd/api
    if [[ "${command_name}" == "app-backend" || "${command_name}" == "server-db-smoke" ]]; then
      go build -o "${runtime_dir}/cove-fake-openai" ./integration/fakeopenai
      go build -o "${runtime_dir}/cove-worker" ./cmd/worker
    fi
  )

  if [[ "${command_name}" == "app-backend" || "${command_name}" == "server-db-smoke" ]]; then
    COVE_E2E_LLM_ADDRESS="127.0.0.1:${e2e_llm_port}" \
      COVE_E2E_LLM_ANSWER="${COVE_E2E_LLM_ANSWER:-Local chat reply persisted.}" \
      "${runtime_dir}/cove-fake-openai" >"${llm_log}" 2>&1 &
    llm_pid=$!
    wait_for_url "deterministic OpenAI-compatible provider" "http://127.0.0.1:${e2e_llm_port}/health" "${llm_pid}" "${llm_log}"
  fi

  if ! server_env "${runtime_dir}/cove-migration" >"${migration_log}" 2>&1; then
    echo "migration failed. Last log lines:" >&2
    tail -n 80 "${migration_log}" >&2 || true
    return 1
  fi

  if [[ "${command_name}" == "app-backend" || "${command_name}" == "server-db-smoke" ]]; then
    server_env "${runtime_dir}/cove-worker" >"${worker_log}" 2>&1 &
    worker_pid=$!
    wait_for_log "Cove worker" "Starting processing" "${worker_pid}" "${worker_log}"
  fi

  server_env "${runtime_dir}/cove-api" >"${api_log}" 2>&1 &
  api_pid=$!
  wait_for_url "Cove API" "${e2e_api_url}/api/health" "${api_pid}" "${api_log}"
  curl --fail --silent --show-error "${e2e_api_url}/api/health" >"${run_artifact_dir}/health.json"

  if [[ "${command_name}" == "server-db-smoke" ]]; then
    (
      cd "${server_dir}"
      COVE_REAL_DB_API_URL="${e2e_api_url}" \
        COVE_REAL_DB_DATABASE_URL="${e2e_database_url}" \
        COVE_REAL_DB_RUN_ID="${e2e_run_id}" \
        COVE_REAL_DB_LLM_URL="${e2e_llm_url}" \
        COVE_REAL_DB_LLM_ANSWER="${COVE_E2E_LLM_ANSWER:-Local chat reply persisted.}" \
        go test -count=1 -v ./integration/realdb
    ) 2>&1 | tee "${server_real_db_log}"
    return
  fi

  if [[ "${command_name}" == "app-backend" ]]; then
    echo "App backend is ready. Keep this command running while Metro and the Simulator flow execute."
    echo "App fixture API URL: ${e2e_api_url}"
    echo "App fixture model base URL: ${e2e_llm_url}"
    wait "${api_pid}"
    return
  fi

  (
    cd "${frontend_dir}"
    VITE_API_BASE_URL="${e2e_api_url}" pnpm exec vite --host 127.0.0.1 --port "${e2e_web_port}" --strictPort
  ) >"${web_log}" 2>&1 &
  web_pid=$!
  wait_for_url "Cove Web" "${e2e_web_url}" "${web_pid}" "${web_log}"

  (
    cd "${frontend_dir}"
    E2E_BASE_URL="${e2e_web_url}" \
      E2E_API_URL="${e2e_api_url}" \
      E2E_RUN_ID="${e2e_run_id}" \
      E2E_ARTIFACT_DIR="${run_artifact_dir}" \
      pnpm test:e2e -- --grep @smoke
  )
}

case "${command_name}" in
  up)
    start_dependencies
    print_environment
    ;;
  app-backend)
    run_smoke
    ;;
  smoke)
    run_smoke
    ;;
  server-db-smoke)
    run_smoke
    ;;
  logs)
    ensure_orbstack
    compose logs --no-color --tail 200 || true
    if [[ -d "${artifact_root}/latest/logs" ]]; then
      for log_file in "${artifact_root}/latest/logs/"*.log; do
        [[ -e "${log_file}" ]] || continue
        echo "===== ${log_file} ====="
        tail -n 200 "${log_file}"
      done
    fi
    ;;
  down)
    ensure_orbstack
    compose down --volumes --remove-orphans
    ;;
  *)
    echo "unknown command: ${command_name}" >&2
    echo "usage: $0 <up|app-backend|server-db-smoke|smoke|logs|down>" >&2
    exit 2
    ;;
esac
