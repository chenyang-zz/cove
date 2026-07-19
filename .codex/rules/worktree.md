# Codex Worktree Isolation

Use Codex-managed worktrees for new product feature implementation so concurrent tasks do not share a checkout or overwrite one another's files. This rule applies to App, Server, and cross-package feature work.

## 1. Task Routing

- A user request to implement a new product feature authorizes one project task in a Codex-managed worktree. When the request begins in a Local task, the Local task remains the coordinator: create the worktree task, wait for its result, review its commit, and archive it after successful handoff.
- Do not create worktrees for read-only exploration, status reports, reviews, committing already completed changes, or rule/document-only updates unless isolation is explicitly requested.
- Use a separate worktree per independently deliverable feature. Do not reuse another running feature's worktree or permanent worktree.
- Start from the requested existing branch. If none is specified, start from the existing `dev` branch. Never use the primary checkout's working-tree state as the starting state unless the user explicitly includes those uncommitted changes.

## 2. Worktree Bootstrap

- Before implementation or dependency-sensitive validation in a new feature worktree, run `bash .codex/scripts/worktree-setup.sh` unless the selected Codex local environment already ran that exact script successfully. Reading repository instructions and performing read-only orientation may happen first.
- The setup script must run inside a linked worktree and must not modify tracked source files. A setup failure blocks implementation until it is fixed or clearly reported.
- Do not copy developer-owned `.env*`, credentials, database URLs, tokens, or local configuration from the primary checkout. Create only run-owned, uncommitted configuration when the feature genuinely requires it.
- In Codex App, a shared Local Environment may use `bash .codex/scripts/worktree-setup.sh` as its setup script. The repository rule remains the fallback when no Local Environment is selected.

## 3. Runtime Isolation

- Every service, Metro process, simulator run, Compose project, database, cache, queue, and test fixture started by the worktree must be run-owned and recorded before use.
- Allocate free local ports at runtime. Do not assume fixed development ports are available and do not stop a process merely because it occupies a preferred port.
- Use a unique Compose project name, container names, volumes, networks, temporary directories, synthetic users, and database names derived from the worktree/task identity. Use the existing OrbStack-based E2E lifecycle for full-stack acceptance.
- Never point feature validation at remote, staging, production, or developer-owned databases. App/Server acceptance uses an isolated local database as required by the E2E rules.
- Register cleanup with `trap` as soon as resources are created. Cleanup may stop or remove only resources whose ownership was established by the current worktree.
- Do not share mutable `node_modules`, build output directories, Metro caches, test artifacts, or generated local configuration between worktrees. Shared immutable package-manager and Go download caches are allowed.

## 4. Completion and Handoff

- Before handoff, run the applicable App, Server, and E2E validation, then stop all run-owned resources and report the cleanup result.
- Create a unique `codex/feature-<slug>-<timestamp>` branch in the worktree before committing. Commit the intended files and record the branch name and commit SHA; never leave the only copy of completed work as an uncommitted worktree diff.
- The coordinating Local task reviews the reported commit and checks that no run-owned resources remain. It must not merge, cherry-pick, push, or delete the feature branch unless the user has authorized that action.
- After successful validation, commit creation, handoff reporting, and runtime cleanup, the coordinator archives the worktree task. Codex archiving provides a recoverable snapshot and removes the managed worktree automatically.
- If validation fails, changes remain uncommitted, cleanup cannot establish resource ownership, or the handoff commit is missing, do not archive or force-remove the worktree. Report the blocker and preserve the worktree for recovery.

## 5. Safe Cleanup

- Prefer Codex task archiving for managed-worktree cleanup. Do not run `git worktree remove --force`, recursively delete a worktree directory, or delete its feature branch as part of normal completion.
- Automatic cleanup applies to the managed worktree and run-owned runtime resources. It does not delete the feature branch or commit; those remain until integration is confirmed.
- A pinned, active, blocked, or permanent worktree is intentionally retained. Clearly report why automatic cleanup did not run.
