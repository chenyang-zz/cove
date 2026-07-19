# Cove App E2E and Server Real-Database Harness

The only current frontend product is the Expo App under `packages/app/mobile/`. Native App acceptance uses the project [`ios-simulator` skill](../.agents/skills/ios-simulator/SKILL.md), with package-owned Maestro flows for adopted deterministic regression scenarios. The root harness provides Server real-database validation and an auxiliary browser regression; it is not a replacement for native App acceptance.

## Current frontend scope

| Product | Location | Required validation |
| --- | --- | --- |
| Cove App | `packages/app/mobile/` | lint/typecheck/Vitest plus an `ios-simulator` skill scenario against the local stack; Maestro when the flow has deterministic regression coverage |

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
- Chat/SSE message creation through a deterministic local OpenAI-compatible provider, Redis, the public API, and PostgreSQL
- Knowledge-base creation and harmless Markdown upload through the public API, local storage, Redis/asynq worker, deterministic embeddings, PostgreSQL, and Elasticsearch search
- Cross-user conversation and message isolation
- Profile normalization and persistence, password rejection and rotation, and old/new password login behavior

The 2026-07-18 interactive iOS Simulator run covers App login, authenticated hydration, SecureStore restoration after process termination, logout, and anonymous state after a second restart. The 2026-07-19 package-owned Maestro runs cover profile/password behavior, deterministic Chat/SSE creation and history restoration, plus protected navigation, keyboard/sheet state, the native back gesture, and SecureStore restoration. Evidence is linked from [`REAL_DATABASE_COVERAGE.md`](./REAL_DATABASE_COVERAGE.md).

Not exercised through the current App yet:

- Registration UI, forced refresh-token rotation, expired/revoked-session behavior, multi-turn/error/interruption chat paths, uploads, and RAG
- Attachment-picker and native Markdown edge cases outside the recorded navigation/session flow
- Worker, scheduler, gateway, Neo4j, and live LLM/MCP provider flows
- These are explicit future suites rather than hidden mocks in the current smoke scenario.
- Live third-party LLM behavior remains out of scope; deterministic local provider behavior is covered without network cost or nondeterministic model output.

## Prerequisites

- OrbStack installed with its `orb` CLI and `orbstack` Docker context available
- The Docker-compatible CLI and Compose plugin provided for OrbStack; Docker Desktop and Colima are not used
- Go, Node.js, pnpm, and the frontend dependencies
- Java 17+ and Maestro CLI for deterministic iOS regression flows; the profile/password workspace is documented in [`packages/app/mobile/e2e/maestro/README.md`](../packages/app/mobile/e2e/maestro/README.md)
- Chromium installed only when running the auxiliary browser smoke:

```bash
pnpm --dir packages/app/frontend exec playwright install chromium
```

## Commands

Run the Server real-database prerequisite first:

```bash
make server-db-smoke
```

This starts run-owned PostgreSQL, Redis, and Elasticsearch services in OrbStack, starts the deterministic local OpenAI-compatible provider, runs the real migration binary, starts the real API and worker, and executes Chat/SSE, document-ingestion, conversation/message, and profile/password persistence scenarios. It does not require the frontend toolchain or a live LLM account.

Run the auxiliary React/Vite smoke with automatic teardown:

```bash
make e2e-smoke
```

`make e2e` currently aliases the auxiliary React/Vite authentication smoke suite. It is not an App result. A run gets a unique Compose project, free local ports, synthetic users, and an artifact directory.

See [`REAL_DATABASE_COVERAGE.md`](./REAL_DATABASE_COVERAGE.md) for the incremental backfill ledger.

After the Server real-database prerequisite passes and the disposable local API, Metro, development build, and explicit Simulator are ready, run the deterministic profile/password App flow with synthetic environment variables:

```bash
IOS_SIMULATOR_UDID=<simulator-udid> \
E2E_RUN_ID=<run-id> \
MAESTRO_EXPO_DEV_CLIENT_URL='exp+cove-mobile://expo-development-client/?url=<encoded-metro-url>' \
MAESTRO_E2E_USERNAME=<synthetic-username> \
MAESTRO_E2E_OLD_PASSWORD=<current-password> \
MAESTRO_E2E_WRONG_PASSWORD=<known-wrong-password> \
MAESTRO_E2E_NEW_PASSWORD=<replacement-password> \
MAESTRO_E2E_NICKNAME=<updated-nickname> \
MAESTRO_E2E_EMAIL=<updated-email> \
make app-mobile-e2e-profile-password
```

The command never provisions or chooses an account. It rotates the supplied password, so use only a disposable fixture in the run-owned local database. Artifacts are stored under `output/ios-simulator/runs/<run-id>/`.

For the chat flow, keep the run-owned App backend active in a separate terminal:

```bash
E2E_RUN_ID=<backend-run-id> \
E2E_PROJECT=<run-owned-project> \
make e2e-app-backend
```

The command starts PostgreSQL, Redis, Elasticsearch, migration, the API, a worker, and the deterministic provider in a run-owned Compose project. It chooses free local ports unless `E2E_*_PORT` values are supplied, prints the resolved API/provider URLs, and records the project and ports in `output/playwright/runs/<backend-run-id>/environment.txt`. In App mode only, the API binds to all host interfaces so the Simulator can reach it. It blocks until interrupted and then removes its host processes, containers, network, and volumes.

Start Metro with the same local API as described below, then run the package-owned flow with new synthetic data:

```bash
IOS_SIMULATOR_UDID=<simulator-udid> \
E2E_RUN_ID=<app-run-id> \
MAESTRO_EXPO_DEV_CLIENT_URL='exp+cove-mobile://expo-development-client/?url=<encoded-metro-url>' \
MAESTRO_E2E_API_URL=<printed-app-backend-api-url> \
MAESTRO_E2E_LLM_BASE_URL=<printed-app-backend-provider-url> \
MAESTRO_E2E_USERNAME=<synthetic-username> \
MAESTRO_E2E_PASSWORD=<synthetic-password> \
MAESTRO_E2E_CHAT_PROMPT=<unique-prompt-at-most-20-characters> \
MAESTRO_E2E_CHAT_ANSWER='Local chat reply persisted.' \
make app-mobile-e2e-chat-persistence
```

The wrapper provisions the disposable account and its default model configuration only through public APIs. The flow logs in, disables knowledge retrieval, asserts the streaming and terminal UI, terminates and relaunches the native process, restores the same history, and verifies the conversation in the drawer.

Run the navigation/native lifecycle flow against the same local backend and Metro processes:

```bash
IOS_SIMULATOR_UDID=<simulator-udid> \
E2E_RUN_ID=<app-run-id> \
MAESTRO_EXPO_DEV_CLIENT_URL='exp+cove-mobile://expo-development-client/?url=<encoded-metro-url>' \
MAESTRO_E2E_API_URL=<printed-app-backend-api-url> \
MAESTRO_E2E_USERNAME=<synthetic-username> \
MAESTRO_E2E_EMAIL=<synthetic-email> \
MAESTRO_E2E_PASSWORD=<synthetic-password> \
MAESTRO_E2E_CHAT_DRAFT=<non-secret-draft> \
MAESTRO_E2E_UNSAVED_NICKNAME=<non-secret-nickname> \
make app-mobile-e2e-native-lifecycle
```

The wrapper provisions its disposable user through the public API. The App rejects an anonymous profile deep link, logs in, exercises keyboard and sheet state, navigates to the real default knowledge base, returns with the native iOS edge gesture, proves mounted draft preservation, cancels an unsaved profile edit, and restores only durable SecureStore state after process termination. The driver captures screenshots, JUnit, sanitized logs, and a full Simulator recording.

## Native App scenario workflow

For every native App flow, load and follow [`.agents/skills/ios-simulator/SKILL.md`](../.agents/skills/ios-simulator/SKILL.md). The skill is the source of truth for discovering the current Expo/dev-client command, choosing an explicit Simulator UDID, building and launching the installed App, controlling foreground UI, diagnosing failures by layer, and capturing evidence. Use Computer Use for exploratory interaction and the package-owned Maestro workspace for adopted deterministic flows.

An App scenario must:

1. Use the disposable local Server/database environment and record the credential-free API base URL.
2. Distinguish native installation, process launch, Metro connection, JavaScript rendering, API reachability, authentication, and App state.
3. Start from a known App state, capture a baseline, and inspect the current UI again after every transition.
4. Assert the user-visible result through the App. Server tests, host-side requests, database rows, and browser smoke are supporting evidence only.
5. Capture screenshots for visual state, recordings for navigation/animation/lifecycle, and sanitized logs for network/authentication behavior.
6. Report the Simulator model, UDID and iOS version; command, bundle ID, Metro state, actions, assertions, evidence paths, first failing layer, and cleanup status.

Computer Use scenarios are reproducible interactive evidence, not automated CI coverage. Maestro scenarios are deterministic local regression coverage; they become CI gates only after a macOS runner owns the documented Simulator build, Metro, local stack, secrets, artifacts, and cleanup lifecycle.

### Isolating the Expo API base URL

An uncommitted `mobile/.env.development.local` may point normal development at another Server. For a disposable local E2E run, do not edit or expose that file. Start Metro from `packages/app/mobile/` with run-owned values and production-mode bundling so Expo statically resolves the intended local API URL:

```bash
EXPO_NO_DOTENV=1 \
EXPO_PUBLIC_API_BASE_URL=http://<mac-lan-ip>:<api-port> \
EXPO_ALLOW_INSECURE_HTTP=true \
pnpm exec expo start --dev-client --lan --port 8081 --clear --no-dev
```

Expo SDK 57 development transforms can merge virtual environment values after the shell environment. The `--no-dev` form above was verified in Cove to keep the run-owned URL in the generated bundle. Confirm the resolved URL without printing credentials, then require a request from the Simulator address in the local API logs; a successful host-side request is not sufficient.

Manage only the disposable dependency stack when debugging:

```bash
make e2e-up
make e2e-app-backend
make e2e-logs
make e2e-down
```

`make e2e-up` and the matching manual `logs`/`down` commands use PostgreSQL `55432`, Redis `56379`, Elasticsearch `59200`, API `58000`, and deterministic provider `58001` by default. Automated `e2e-app-backend`, `server-db-smoke`, and browser smoke runs instead use a unique Compose project and free ports unless explicitly overridden. Always pass the URLs printed by the active App-backend run to Metro and the native wrapper.

## Lifecycle and artifacts

- `e2e/scripts/e2e.sh` is the lifecycle source of truth.
- The script starts OrbStack when necessary and always runs Compose through `docker --context orbstack`; the Docker CLI is only the OrbStack-compatible client, not the container runtime.
- The automated commands wait for Compose health, run the real migration, and wait for `/api/health`. The Server and App-backend commands also start a run-owned worker and deterministic provider; worker readiness and process logs are kept separate. The browser command additionally starts Vite and runs Playwright serially. `e2e-app-backend` keeps the same local services alive for a native Simulator run.
- Every automated command stops host processes and removes run-owned containers and volumes on success, failure, timeout, or interruption.
- Set `E2E_KEEP_ENV=1` only for local diagnosis. Clean it later with the same `E2E_PROJECT` value.
- The run-owned project, ports, and credential-free URLs are recorded in `environment.txt`. API, worker, migration, provider, Compose, and backend scenario logs plus browser traces, screenshots, video, and the HTML report are written below `output/playwright/runs/<run-id>/`.
- `output/playwright/latest` points to the most recent run.

The harness never reads the normal Server config or the persistent development Compose stack. Test credentials and data are local, synthetic, and run-owned.
