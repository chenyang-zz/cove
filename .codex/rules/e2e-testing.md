# Cove End-to-End Testing Rules

These rules apply when adding, changing, running, or diagnosing tests that cross a real Cove process or product boundary. Read `.codex/rules/architecture.md` first, then read the frontend and backend rules for every surface exercised by the scenario.

## 1. Definition and Current State

- Cove currently has one frontend product: the Expo App under `packages/app/mobile/`. App acceptance means exercising this native application, with iOS Simulator as the primary environment.
- The React/Vite client under `packages/app/frontend/` and the Wails shell are not current frontend product surfaces. Their existing tests are auxiliary or legacy regression evidence and should be run only when explicitly relevant; do not report them as App acceptance.
- A Server real-database scenario is required backend evidence and a prerequisite for cross-boundary acceptance, but it is never evidence that the App UI, App session state, navigation, rendering, or platform behavior passed.
- An end-to-end test must exercise a user-visible or externally observable flow across real boundaries, such as browser or Simulator UI → HTTP/SSE → server logic → persistence or queue processing.
- Vitest component tests, Go unit tests, `httptest` router tests, snapshots, and a successful build are valuable but are not end-to-end proof by themselves.
- Cove currently has a Server real-database harness under `packages/server/integration/realdb/` and an auxiliary React/Vite Playwright smoke under `packages/app/frontend/`. Workspace orchestration lives under `e2e/`; both use isolated local infrastructure, but neither is native App UI acceptance.
- Native App acceptance is executed through the project `.agents/skills/ios-simulator/` skill. These runs are skill-driven Simulator scenarios with reproducible evidence; they are not a deterministic CI suite unless the flow is also encoded in XCUITest, Maestro, Detox, or another project-owned regression framework.
- Current Server real-database coverage additionally includes profile persistence, password rejection and rotation, conversation CRUD persistence, message-history pagination, and cross-user conversation/message isolation. Do not report those Server-only scenarios as App coverage.
- The existing `packages/server/deployments/docker-compose.yml` is a persistent development/production-like stack. Do not use it for destructive E2E setup or cleanup unless the run is explicitly isolated by project name, ports, data directories, and volumes.
- Browser flows outside the implemented Playwright suite and skill-driven Simulator flows remain interactive E2E evidence until encoded in a deterministic regression framework. Include the exact setup, actions, assertions, and artifacts when reporting them.

## 2. Test Selection and Coverage

- For a user-visible feature, define its native App acceptance flow first. Use Server scenarios to establish backend correctness, then exercise the behavior through the Expo App in iOS Simulator.
- A feature is not frontend-complete when only `make server-db-smoke`, Go tests, direct HTTP calls, database assertions, Vitest, or the auxiliary browser smoke pass. The Expo App must exercise the user-visible behavior or the missing Simulator validation must be reported explicitly.
- Add E2E coverage for behavior whose main risk lies between components: authentication/session rotation, authorization isolation, client/server contract changes, SSE ordering, async worker completion, native navigation, uploads, gateway delivery, or recovery after dependency failure.
- Keep business edge cases in unit and integration tests. E2E tests should cover a small number of representative happy paths and high-value failure boundaries rather than reimplementing the entire lower-level suite.
- Every E2E scenario must name the contract it proves, the processes it starts, the external systems it replaces, and the observable completion condition.
- A cross-package API or SSE change is not complete until the relevant server behavior and each affected client surface have been exercised or an explicit validation gap has been reported.
- Prioritize a smoke suite containing health readiness, registration/login, authenticated hydration, refresh-token rotation, one persistent resource flow, ownership isolation between two users, and one controlled streaming-chat flow.
- Put long-running RAG ingestion, worker retry, gateway/provider, and platform matrix scenarios in a fuller scheduled suite unless the change directly affects them.

## 3. Harness Ownership and Layout

- Keep cross-package orchestration at the `cove/` monorepo root. Keep package-specific drivers, fixtures, and assertions inside the package that owns them.
- Keep App drivers and assertions with `packages/app/mobile/` or the project iOS Simulator skill. Do not place App UI assertions in the Server package. Keep the existing browser driver under `packages/app/frontend/e2e/` labeled as auxiliary regression coverage.
- Use the implemented workspace-root Make targets: `server-db-smoke`, `e2e-up`, `e2e-smoke`, `e2e`, `e2e-logs`, and `e2e-down`. Keep `e2e/scripts/e2e.sh` as their lifecycle source of truth.
- Keep one documented source of truth for service startup, health checks, environment variables, test data setup, artifact paths, and teardown. Do not duplicate lifecycle logic across shell scripts, CI YAML, and test code.
- Prefer a thin orchestration layer that calls package-native tools. Keep App and Server dependency manifests and test configurations package-local.
- Name scenarios by product behavior, not implementation detail, for example `auth-refresh-rotation`, `chat-stream-completes`, or `knowledge-upload-becomes-ready`.

## 4. Local Environment Lifecycle

- Use OrbStack as Cove's local E2E container runtime. Run Compose explicitly through the `orbstack` Docker context; do not depend on Docker Desktop, Colima, or whichever Docker context happens to be current.
- Final cross-frontend/backend E2E acceptance must use local, disposable infrastructure. Never point automated tests at a remote, production, staging, or developer-owned database or at a service containing persistent user data.
- Give each run a unique Compose project and run identifier. Isolate ports, databases, Redis namespaces, Elasticsearch indices, Neo4j databases, object-storage prefixes, test users/resources, and artifact directories where the dependency supports it.
- Wait for explicit readiness: container health, `/api/health`, a successful migration check, and worker availability where required. A started process or open TCP port alone is not sufficient.
- Run the real server migration path against the run-owned local database before scenarios that depend on persistence. Seed only the minimal deterministic state required by the test.
- Teardown must run after success, failure, timeout, or interruption. Remove the run-owned containers, networks, volumes, temporary files, processes, and test data without deleting resources outside the run identifier.
- Reusing a local stack is allowed only when reset behavior is deterministic and documented. Provide a clean-reset path for schema drift, stale queue messages, orphaned indices, and corrupted fixtures.
- Do not run scenarios concurrently against one shared local schema or queue. Parallel execution requires per-worker isolation; otherwise make the suite explicitly serial.

## 5. Test Data, Accounts, and Secrets

- Generate unique user accounts and resource names from the run identifier. Do not depend on execution order, pre-existing IDs, or another test's leftovers.
- Use at least two users for ownership and authorization scenarios. Assert both the allowed operation and the cross-user denial at the public boundary.
- Prefer setup through public APIs when setup behavior is part of the contract. A test-only bootstrap path may seed expensive prerequisites, but it must be narrow, disabled outside test environments, and must not bypass the behavior being asserted.
- Keep credentials in test-only environment files or CI secrets. Never copy database passwords, remote URLs, API keys, JWTs, refresh tokens, or provider payloads from external notes into Cove rules, fixtures, logs, or screenshots.
- Redact tokens and private content from test output and artifacts. Use synthetic documents, images, prompts, and webhook payloads.

## 6. Auxiliary Browser and Legacy Desktop Flows

- React/Vite under `packages/app/frontend/` is auxiliary regression coverage, not the current App frontend. Report its Playwright result as “auxiliary browser smoke,” never as App E2E.
- When browser automation is adopted, prefer Playwright as the default Web E2E driver unless the repository deliberately selects another maintained tool. Add it as an explicit project dependency and configuration, not as an assumption based on a transitive Vitest package.
- Exercise the browser against the real Cove API base URL supplied through `VITE_API_BASE_URL`. Do not intercept the API under test or replace the entire server with browser route mocks.
- Prefer accessible roles, labels, and visible product text. Add stable product-intent test IDs only when semantic selectors cannot uniquely express the action; never target generated classes or DOM structure.
- Cover session persistence, one refresh-and-retry cycle, logout/expired-session behavior, SSE incremental rendering, and browser navigation when those contracts change.
- Wails-only behavior requires the real desktop shell or an explicit bridge-focused test. Browser-only success does not prove native window, menu, file-dialog, or Go binding behavior.

## 7. Expo and Native Mobile Flows

- Before running or diagnosing an App scenario, load and follow `.agents/skills/ios-simulator/SKILL.md`. Treat that skill and its `references/launch-and-diagnose.md` as the execution protocol for discovery, build, launch, interaction, diagnosis, evidence, and handoff; do not create a separate ad hoc Simulator workflow in this rule.
- Establish the skill baseline before interaction: preserve `git status --short`, identify the canonical Expo/dev-client command, bundle ID, URL scheme, Metro state, resolved API base URL, and local backend, then select and record one explicit Simulator UDID.
- Keep native installation, process launch, Metro connection, JavaScript rendering, local API reachability, authentication, and App state as separate diagnostic layers. A successful build, an installed development-client shell, or a host-side API request is not App acceptance.
- Resolve and verify the App API base URL for every disposable run. If Expo SDK 57 development transforms allow an uncommitted local environment file to override run-owned shell values, preserve the developer file and launch Metro with run-owned `EXPO_PUBLIC_API_BASE_URL`, `EXPO_NO_DOTENV=1`, and `--no-dev`; confirm both the generated bundle value and a request from the Simulator address in local API logs.
- Use Codex Computer Use as the default foreground path for a one-off App flow. Read the current Simulator state before the first action and after every navigation, animation, keyboard transition, alert, sheet, or lifecycle transition; prefer current accessibility targets and stable `testID` values over coordinates.
- Use a development build for flows that require native modules. Expo Go is not valid evidence for native-module behavior that Cove does not support there.
- Drive controls through accessibility semantics and stable English kebab-case `testID` values from `.codex/rules/frontend.md`. Do not navigate by screen coordinates when a semantic selector is available.
- Do not use a deep link to skip authentication, navigation, data creation, or another behavior that the scenario is intended to prove. Deep links are acceptable only for unrelated setup.
- Validate native navigation lifecycle, keyboard behavior, sheets, protected routes, session persistence, and streaming updates when affected. Capture a baseline and relevant final screenshot; capture a recording when intermediate motion, navigation, keyboard, background/foreground, or lifecycle behavior is part of the contract.
- For network or authentication behavior, require the App-visible result plus relevant sanitized logs. A host-side `curl`, database row, Server test, or resting screenshot alone is insufficient.
- Use a project-owned XCUITest, Maestro, Detox, or equivalent framework when the scenario must become deterministic CI regression coverage. Do not introduce one only to satisfy a single interactive validation; first document the missing capability, Expo SDK 57/React Native 0.86 compatibility, CI requirements, and maintenance cost.

## 8. Server, SSE, Workers, and Persistence

- Start the API, worker, scheduler, or gateway only when the scenario needs them. Keep process logs separate and label them with the run identifier.
- Assert public HTTP status, Cove response envelope, authentication behavior, and persisted effects. Do not make an E2E test pass solely by inspecting an internal Go value.
- For SSE, assert the meaningful event protocol and terminal state rather than arbitrary chunk boundaries. Use bounded waits for `meta`, `think`, `tool_call`, `tool_result`, `token`, `done`, and `error` events relevant to the scenario.
- For async ingestion or delivery, poll the public status API or durable observable state with a deadline and useful diagnostics. Do not use fixed sleeps as the synchronization mechanism.
- Verify retry and idempotency flows with duplicate input or a controlled transient failure when the changed behavior owns those guarantees.
- Keep database assertions read-only and secondary to public assertions. Direct queries may diagnose persistence or verify an invariant that no public API exposes, but they must use the run-owned local database and user/resource scope.

## 9. External Providers and Failure Control

- Do not call live LLM, MCP, email, webhook, object-storage, or messaging-provider accounts in the default E2E suite.
- Replace external providers at their network boundary with deterministic local servers or project-owned test adapters while keeping Cove's internal HTTP, queue, persistence, SSE, and mapping paths real.
- Script provider responses, latency, malformed payloads, disconnects, and retryable failures explicitly. Assert whether Cove must fail open or fail closed.
- Maintain a separate opt-in contract or sandbox suite for real providers when needed. Require explicit credentials, bounded cost, rate-limit awareness, cleanup, and a clear label that results may be provider-dependent.

## 10. Synchronization and Flake Policy

- Wait on observable conditions with deadlines: health status, UI state, SSE terminal events, database state, task status, or process exit. Never hide a race with an unconditional sleep.
- Make timeouts configurable by environment while keeping a finite default. Failure output must identify the condition, elapsed time, last observed state, and relevant process logs.
- A retry may collect diagnostic evidence, but it must not convert a flaky test into a passing gate. Record first-attempt failure and fix or quarantine the scenario with an owner and reason.
- Control clocks, randomness, provider output, and generated identifiers when they influence assertions. Avoid exact timing assertions unless performance is the contract under test.
- Establish Cove performance baselines from repeatable Cove runs. Do not copy duration or speedup numbers from another project into acceptance criteria.

## 11. Execution Tiers and Reporting

- Pull requests should run a deterministic smoke subset sized for normal development. Run broader infrastructure, native, provider-contract, and platform-matrix suites on scheduled or explicitly requested jobs.
- Run focused unit and integration tests before E2E so failures are localized quickly. E2E success does not replace App or Server package validation.
- Before App/Server E2E, complete the backend local real-database scenario required by `.codex/rules/backend.md`. Then run Expo lint, typecheck, and tests plus the real user flow in iOS Simulator.
- Report the focused Server database scenario and the App result as separate stages. “Server passed” must never be summarized as “frontend passed.”
- For every skill-driven App result, report the Simulator model, UDID and iOS version; build/launch command; bundle ID and Metro state; credential-free API base URL; initial state and interaction sequence; screenshot/recording/log paths; assertions; first failing layer when applicable; and cleanup status.
- On failure, preserve the smallest useful evidence set: orchestrator output, service logs, request correlation IDs, browser trace/screenshot/video, or Simulator screenshot/recording and device logs.
- Do not retain secrets or full private payloads in artifacts. Apply explicit artifact retention and size limits in CI.
- Report App and Server results separately, then report the cross-boundary scenario result. List services exercised, services replaced by fakes, scenarios skipped, retries, cleanup status, and any environment not tested.

## 12. Applying the External Testing Notes

- Treat the supplied external `testing.md` as a source of reusable lifecycle ideas, not Cove configuration.
- Preserve its useful principles: local disposable dependencies, explicit startup and teardown, health checks, deterministic reset, serial execution for shared state, documented troubleshooting, and measured local performance.
- Do not copy its Python, pytest, asyncpg, RLS-role, database-name, credential, port, image, remote-database, or timing details. Cove uses Go, Gin, GORM, Redis/asynq, Elasticsearch, Neo4j, React/Vite, Expo, and iOS Simulator.
- Promote a concrete Compose service, port, Make target, framework, fixture, or benchmark to a Cove rule only after it exists in this workspace and has been verified here.
