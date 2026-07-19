#!/usr/bin/env bash

set -euo pipefail

usage() {
    echo "Usage: $0 [--check]" >&2
}

check_only=false
if [[ "${1:-}" == "--check" ]]; then
    check_only=true
    shift
fi
if [[ "$#" -ne 0 ]]; then
    usage
    exit 64
fi

workspace_root="$(git rev-parse --show-toplevel)"
git_dir="$(git rev-parse --path-format=absolute --git-dir)"
git_common_dir="$(git rev-parse --path-format=absolute --git-common-dir)"

if [[ "$git_dir" == "$git_common_dir" && "${COVE_ALLOW_PRIMARY_SETUP_CHECK:-false}" != "true" ]]; then
    echo "Refusing to initialize the primary checkout; create a Codex-managed worktree first." >&2
    exit 2
fi

required_files=(
    "go.work"
    "packages/app/go.mod"
    "packages/server/go.mod"
    "packages/app/frontend/pnpm-lock.yaml"
    "packages/app/mobile/pnpm-lock.yaml"
)

for required_file in "${required_files[@]}"; do
    if [[ ! -f "$workspace_root/$required_file" ]]; then
        echo "Missing required workspace file: $required_file" >&2
        exit 1
    fi
done

if ! command -v go >/dev/null 2>&1; then
    echo "Go is required to initialize the Cove worktree." >&2
    exit 1
fi

run_pnpm() {
    if command -v pnpm >/dev/null 2>&1; then
        pnpm "$@"
        return
    fi
    if command -v corepack >/dev/null 2>&1; then
        corepack pnpm "$@"
        return
    fi
    echo "pnpm or Corepack is required to initialize the Cove worktree." >&2
    exit 1
}

if [[ "$check_only" == "true" ]]; then
    go version
    run_pnpm --version
    echo "Cove worktree setup prerequisites are available."
    exit 0
fi

initial_status="$(git -C "$workspace_root" status --porcelain)"

(
    cd "$workspace_root/packages/app"
    GOWORK=off go mod download
)
(
    cd "$workspace_root/packages/server"
    GOWORK=off go mod download
)
run_pnpm --dir "$workspace_root/packages/app/frontend" install --frozen-lockfile
run_pnpm --dir "$workspace_root/packages/app/mobile" install --frozen-lockfile

final_status="$(git -C "$workspace_root" status --porcelain)"
if [[ "$final_status" != "$initial_status" ]]; then
    echo "Worktree setup changed tracked or untracked workspace files; inspect before implementation." >&2
    git -C "$workspace_root" status --short >&2
    exit 1
fi

echo "Cove worktree dependencies are ready."
