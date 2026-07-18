# Cove App E2E and Server Real-Database Harness

The only current frontend product is the Expo App under `packages/app/mobile/`. Native App acceptance uses the project [`ios-simulator` skill](../.agents/skills/ios-simulator/SKILL.md). The root harness provides Server real-database validation and an auxiliary browser regression; it is not a replacement for the skill-driven App scenario.

## Current frontend scope

| Product | Location | Required validation |
| --- | --- | --- |
| Cove App | `packages/app/mobile/` | lint/typecheck/Vitest plus an `ios-simulator` skill scenario against the local stack |

The React/Vite client and Wails shell are auxiliary or legacy surfaces, not current frontend products. `make e2e-smoke` exercises React/Vite only and must not be reported as App E2E.

`make server-db-smoke` validates Server behavior and persistence. It is a prerequisite for App/Server E2E, but it does not prove any App screen, navigation, SecureStore session, native module, or platform behavior.

## Current smoke boundary

Current auxiliary browser coverage:

- Chromium and the React/Vite client from `packages/app/frontend/`
- Cove API and database migration binaries built from `packages/server/`
- PostgreSQL, Redis, and Elasticsearch in disposable Compose containers hosted by OrbStack
- Browser session persistence, access-token refresh, and refresh-token rotation

Current Server real-database coverage:

- Backend conversation creation, rename, listing and deletion through the public API
- Message persistence and pagination through the real PostgreSQL repository and public API
- Cross-user conversation and message isolation
- Profile normalization and persistence, password rejection and rotation, and old/new password login behavior

The 2026-07-18 interactive iOS Simulator run now covers App login, authenticated hydration, SecureStore restoration after process termination, logout, and anonymous state after a second restart. Evidence is linked from [`REAL_DATABASE_COVERAGE.md`](./REAL_DATABASE_COVERAGE.md).

Not exercised through the current App yet:

- Registration UI, forced refresh-token rotation, expired/revoked-session behavior, profile/password editing, chat creation/history, uploads, and RAG
- Expo Router gestures, keyboard/sheet lifecycle, and native-module behavior outside the recorded authentication flow
- Worker, scheduler, gateway, Neo4j, and live LLM/MCP provider flows
- These are explicit future suites rather than hidden mocks in the current smoke scenario.
- Chat-generated message persistence is still pending because the current backend scenario seeds messages through the owning repository; it does not call a live LLM.

## Prerequisites

- OrbStack installed with its `orb` CLI and `orbstack` Docker context available
- The Docker-compatible CLI and Compose plugin provided for OrbStack; Docker Desktop and Colima are not used
- Go, Node.js, pnpm, and the frontend dependencies
- Chromium installed only when running the auxiliary browser smoke:

```bash
pnpm --dir packages/app/frontend exec playwright install chromium
```

## Commands

Run the Server real-database prerequisite first:

```bash
make server-db-smoke
```

This starts run-owned PostgreSQL, Redis, and Elasticsearch services in OrbStack, runs the real migration binary, starts the real API, and executes the conversation/message plus profile/password persistence and ownership scenarios. It does not require the frontend toolchain.

Run the auxiliary React/Vite smoke with automatic teardown:

```bash
make e2e-smoke
```

`make e2e` currently aliases the auxiliary React/Vite authentication smoke suite. It is not an App result. A run gets a unique Compose project, free local ports, synthetic users, and an artifact directory.

See [`REAL_DATABASE_COVERAGE.md`](./REAL_DATABASE_COVERAGE.md) for the incremental backfill ledger.

## Native App scenario workflow

For every native App flow, load and follow [`.agents/skills/ios-simulator/SKILL.md`](../.agents/skills/ios-simulator/SKILL.md). The skill is the source of truth for discovering the current Expo/dev-client command, choosing an explicit Simulator UDID, building and launching the installed App, controlling foreground UI with Computer Use, diagnosing failures by layer, and capturing evidence.

An App scenario must:

1. Use the disposable local Server/database environment and record the credential-free API base URL.
2. Distinguish native installation, process launch, Metro connection, JavaScript rendering, API reachability, authentication, and App state.
3. Start from a known App state, capture a baseline, and inspect the current UI again after every transition.
4. Assert the user-visible result through the App. Server tests, host-side requests, database rows, and browser smoke are supporting evidence only.
5. Capture screenshots for visual state, recordings for navigation/animation/lifecycle, and sanitized logs for network/authentication behavior.
6. Report the Simulator model, UDID and iOS version; command, bundle ID, Metro state, actions, assertions, evidence paths, first failing layer, and cleanup status.

These are reproducible interactive E2E scenarios. When a flow must become a deterministic CI gate, add it to a project-owned native regression framework; do not describe Computer Use interaction alone as automated CI coverage.

### Isolating the Expo API base URL

An uncommitted `mobile/.env.development.local` may point normal development at another Server. For a disposable local E2E run, do not edit or expose that file. Start Metro from `packages/app/mobile/` with run-owned values and production-mode bundling so Expo statically resolves the intended local API URL:

```bash
EXPO_NO_DOTENV=1 \
EXPO_PUBLIC_API_BASE_URL=http://<mac-lan-ip>:<api-port> \
EXPO_ALLOW_INSECURE_HTTP=true \
pnpm exec expo start --dev-client --lan --port 8081 --clear --no-dev
```

Expo SDK 57 development transforms can merge virtual environment values after the shell environment. The `--no-dev` form above was verified in Cove to keep the run-owned URL in the generated bundle. Confirm the resolved URL without printing credentials, then require a request from the Simulator address in the local API logs; a successful host-side request is not sufficient.

Manage only the disposable dependency stack on stable default ports when debugging:

```bash
make e2e-up
make e2e-logs
make e2e-down
```

The manual stack uses PostgreSQL `55432`, Redis `56379`, and Elasticsearch `59200` by default. Override `E2E_*_PORT` or `E2E_PROJECT` when required.

## Lifecycle and artifacts

- `e2e/scripts/e2e.sh` is the lifecycle source of truth.
- The script starts OrbStack when necessary and always runs Compose through `docker --context orbstack`; the Docker CLI is only the OrbStack-compatible client, not the container runtime.
- Both automated commands wait for Compose health, run the real migration, and wait for `/api/health`. The backend command then runs its Go scenario; the browser command additionally starts Vite and runs Playwright serially.
- Every automated command stops host processes and removes run-owned containers and volumes on success, failure, timeout, or interruption.
- Set `E2E_KEEP_ENV=1` only for local diagnosis. Clean it later with the same `E2E_PROJECT` value.
- API, migration, Compose, and backend scenario logs plus browser traces, screenshots, video, and the HTML report are written below `output/playwright/runs/<run-id>/`.
- `output/playwright/latest` points to the most recent run.

The harness never reads the normal Server config or the persistent development Compose stack. Test credentials and data are local, synthetic, and run-owned.
