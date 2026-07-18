# Cove Real Database Coverage

The only current frontend product is the Expo App under `packages/app/mobile/`. Native acceptance is executed through [`.agents/skills/ios-simulator/SKILL.md`](../.agents/skills/ios-simulator/SKILL.md) and tracked separately from auxiliary browser and Server persistence evidence.

## App acceptance coverage

| App flow | Real boundary | Command/evidence | Status | Remaining gap |
| --- | --- | --- | --- | --- |
| Authentication and SecureStore session restoration | iOS Simulator → Expo App → API → PostgreSQL | `output/ios-simulator/runs/20260718-auth-session/evidence/` | Partially covered | Add App registration, forced refresh-token rotation, and expired/revoked-session behavior. |
| Profile and password UI | iOS Simulator → Expo profile screen → API → PostgreSQL | `ios-simulator` skill evidence | Not yet recorded | Edit profile, show wrong-password feedback, change password, logout, and log in with the new password. |
| Persistent chat and SSE | iOS Simulator → Expo chat/SSE → API → PostgreSQL/Redis | `ios-simulator` skill evidence | Not yet recorded | Create chat, render deterministic stream, relaunch, and restore history. |
| Navigation and native lifecycle | iOS Simulator → Expo Router/native stack | `ios-simulator` skill evidence | Not yet recorded | Validate protected routes, gestures, keyboard, sheets, and native modules. |

### Recorded App runs

#### 2026-07-18 authentication session lifecycle

- Environment: run-owned OrbStack PostgreSQL, Redis, and Elasticsearch; migrated local API; Expo development client connected to a local Metro bundle.
- Simulator: iPhone Air, iOS 26.3, UDID `09864DA9-634C-4FA0-8900-4436A372F656`, portrait; bundle ID `io.github.chenyangzz.cove.mobile`.
- Covered flow: synthetic account fixture → App login → authenticated `/api/auth/me` hydration → process termination and SecureStore restoration without another login → App logout → second process restart remaining anonymous.
- Network assertions: the App login and hydration requests reached the local API with HTTP 200; the authenticated restart issued `/api/auth/me` without a new `/api/auth/login`; the post-logout restart issued neither request.
- Evidence: `output/ios-simulator/runs/20260718-auth-session/evidence/05-login-success.png`, `06-session-restored.png`, `09-logout-complete.png`, `11-logout-persists-settled.png`, and the logout-restart lifecycle recording `12-logout-restart-lifecycle.mp4`.
- Automation status: interactive skill evidence, not a deterministic CI test. Password entry and logout required user handoff after Computer Use timed out while reading the Simulator accessibility tree.
- Cleanup: Metro and the local API were stopped; the `cove-e2e` containers, network, and volumes were removed through `make e2e-down`.
- Remaining coverage: registration through the App UI, a forced access-token expiry with refresh-token rotation, and expired/revoked-session rejection.

## Auxiliary browser coverage

| Flow | Boundary | Command | Status |
| --- | --- | --- | --- |
| React/Vite authentication and session rotation | Chromium → `packages/app/frontend` → API → PostgreSQL | `make e2e-smoke` | Covered as auxiliary regression only |

## Server real-database coverage

The table below tracks behavior exercised through the workspace-owned OrbStack environment after the real server migration path. It is backend evidence, not App acceptance. A green unit test or mocked router test does not mark an item complete here.

| Area | Real boundary | Command | Status | Remaining gap |
| --- | --- | --- | --- | --- |
| Migration and API readiness | Migration binary → PostgreSQL; API → PostgreSQL/Redis/Elasticsearch | `make server-db-smoke`, `make e2e-smoke` | Covered | Add explicit schema compatibility checks when destructive migrations are introduced. |
| Authentication and session rotation | Browser → React/Vite → API → PostgreSQL | `make e2e-smoke` | Covered | Add expired and revoked session failure scenarios. |
| Conversation persistence | API create/list/rename/delete → PostgreSQL | `make server-db-smoke` | Covered | Add group-conversation membership scenarios. |
| Message history and ownership | PostgreSQL repository seed → API pagination/ownership → PostgreSQL | `make server-db-smoke` | Partially covered | Exercise message creation through deterministic local chat/SSE provider responses. |
| Profile and password changes | API profile/password/login/me → PostgreSQL | `make server-db-smoke` | Covered | Add avatar/object-storage coverage and explicit refresh-token revocation if password changes adopt that policy. |
| Model, agent, persona, MCP, skill, and tool configuration | API → PostgreSQL | — | Missing | Add representative CRUD and cross-user isolation flows. |
| Knowledge bases and document metadata | API/worker → PostgreSQL/object storage | — | Missing | Add upload, queue completion, and ownership scenarios. |
| RAG indexing and retrieval | API/worker → PostgreSQL/Redis/Elasticsearch | — | Missing | Add deterministic ingestion, retrieval, and cleanup. |
| Worker and scheduler tasks | API/scheduler → Redis/asynq → worker → persistence | — | Missing | Add retry, idempotency, and terminal-state scenarios. |
| Gateway delivery | Gateway → queue/outbox → provider boundary | — | Missing | Add deterministic local provider and duplicate-event scenarios. |
| Expo App cross-boundary acceptance | `ios-simulator` skill → Expo App → API → persistence | Skill scenario evidence | Not yet recorded | Execute and record the native App flows listed above after each backend prerequisite passes. |

## App acceptance backfill order

1. Expo authentication and SecureStore session lifecycle in iOS Simulator.
2. Expo profile/password flow against the local database environment.
3. Expo persistent chat and deterministic Chat/SSE creation.
4. Expo navigation, keyboard, sheet, and native-module lifecycle.

Each App item must follow the `ios-simulator` skill and record an explicit Simulator UDID, the App-visible assertions, screenshot or recording paths, relevant sanitized logs, the first failing layer when applicable, and cleanup status. Skill-driven Computer Use is interactive evidence; mark deterministic automation separately only after a native regression framework owns the scenario.

## Server persistence backfill order

1. Representative configuration CRUD with cross-user isolation.
2. Deterministic Chat/SSE message creation using a local provider stub.
3. Knowledge upload and worker completion.
4. RAG retrieval, retries, and gateway delivery.

Each new row must use run-unique synthetic data, public assertions as the primary proof, direct database assertions only for otherwise invisible invariants, and scoped cleanup through the shared E2E lifecycle.
