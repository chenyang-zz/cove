# Cove Real Database Coverage

The only current frontend product is the Expo App under `packages/app/mobile/`. Native acceptance is executed through [`.agents/skills/ios-simulator/SKILL.md`](../.agents/skills/ios-simulator/SKILL.md) and tracked separately from auxiliary browser and Server persistence evidence.

## App acceptance coverage

| App flow | Real boundary | Command/evidence | Status | Remaining gap |
| --- | --- | --- | --- | --- |
| Authentication and SecureStore session restoration | iOS Simulator → Expo App → API → PostgreSQL | `output/ios-simulator/runs/20260718-auth-session/evidence/` | Partially covered | Add App registration, forced refresh-token rotation, and expired/revoked-session behavior. |
| Profile and password UI | iOS Simulator → Expo profile screen → API → PostgreSQL | `make app-mobile-e2e-profile-password` | Covered locally | Promote the package-owned Maestro flow to a macOS CI gate and add avatar/object-storage coverage. |
| Persistent chat and SSE | iOS Simulator → Expo chat/SSE → API → deterministic provider/Redis/PostgreSQL | `make app-mobile-e2e-chat-persistence` | Covered locally | Add multi-turn context, cancellation/error persistence, and tool event rendering. |
| Navigation and native lifecycle | iOS Simulator → Expo Router/native stack → API/PostgreSQL | `make app-mobile-e2e-native-lifecycle` | Covered locally | Promote to a macOS CI gate; add registration Password AutoFill and attachment-picker/native Markdown scenarios. |
| Knowledge document upload and parsing | iOS Simulator → native DocumentPicker → Expo App → API/Redis worker/PostgreSQL/local storage/Elasticsearch | `make app-mobile-e2e-knowledge-upload` | Covered locally | Add cancellation, unsupported/oversized files, provider failure/retry, cross-user isolation, and Android coverage. |

### Recorded App runs

#### 2026-07-19 navigation and native lifecycle

- Environment: run-owned OrbStack PostgreSQL, Redis, and Elasticsearch; real migration and API; Expo development client connected to run-owned Metro with `EXPO_NO_DOTENV=1` and the LAN-reachable local API URL.
- Simulator: iPhone Air, iOS 26.3, UDID `09864DA9-634C-4FA0-8900-4436A372F656`, portrait; bundle ID `io.github.chenyangzz.cove.mobile`.
- Covered flow: public-API disposable account fixture → anonymous profile deep link rejected by `Stack.Protected` → App login → keyboard-backed chat draft → drawer navigation to the Server-created default knowledge base → iOS interactive-pop gesture → mounted chat draft preserved → profile edit sheet with keyboard → unsaved nickname cancelled → return to chat → native process termination/relaunch → SecureStore hydration → page-local draft absent and cancelled nickname still absent.
- App-visible assertions: authenticated screens never rendered for the anonymous deep link; the default knowledge base and count rendered; the draft survived push/pop but not process termination; the cancelled nickname never replaced the username; the relaunched App restored the authenticated profile without a second login.
- Network assertions: host-side fixture registration returned 200; Simulator-originated login, `/api/auth/me`, conversation list, knowledge-base list, post-restart `/api/auth/me`, and post-restart conversation list all returned 200 from `192.168.2.142`. No profile update request was issued for the cancelled edit.
- Server prerequisite: `make server-db-smoke` passed all three real-database scenarios against an independently migrated run-owned stack before the App flow.
- Automation: package-owned Maestro 2.6.1 flow, `1/1` passed in 71 seconds; the complete Simulator recording is 83 seconds. Passwords are redacted from retained text artifacts and generated command JSON is deleted.
- Evidence: `output/ios-simulator/runs/20260719-native-lifecycle-r7/maestro-junit.xml`, `evidence/native-lifecycle.mp4`, and `evidence/screenshots/01-protected-route-rejected.png` through `07-secure-session-restored.png`; supporting API logs are under `output/playwright/runs/20260719-native-lifecycle-backend/logs/`.
- Harness diagnosis before the passing run: cold Metro bundling exceeded the first onboarding wait; iOS Password AutoFill made registration UI automation nondeterministic, so registration remains an explicit separate gap and the fixture moved to the public API; the real knowledge endpoint creates a default knowledge base rather than returning an empty list; Maestro keyboard dismissal on the transparent profile Modal also tapped the scrim. The final flow encodes each verified product contract without changing App business logic.
- Cleanup: Metro, the local API, and the deterministic provider were stopped; run-owned containers, network, and volumes were removed by the root lifecycle.

#### 2026-07-19 persistent chat and SSE lifecycle

- Environment: run-owned OrbStack PostgreSQL, Redis, and Elasticsearch; real migration and API; deterministic local OpenAI-compatible provider at the Server boundary; Expo development client connected to run-owned Metro without reading the normal dotenv file.
- Simulator: iPhone Air, iOS 26.3, UDID `09864DA9-634C-4FA0-8900-4436A372F656`, portrait; bundle ID `io.github.chenyangzz.cove.mobile`.
- Covered flow: public-API synthetic account/model fixture → App login → knowledge retrieval disabled → App sends a unique prompt → visible streaming state → deterministic terminal answer → native process termination/relaunch → SecureStore hydration → conversation and two-message history restoration → conversation drawer discovery.
- App-visible assertions: `Cove 正在回复` appeared during the response; the prompt, `Local chat reply persisted.`, and generated title rendered before restart; the same values rendered after restart; the drawer listed the restored conversation.
- Network assertions: Simulator-originated login, `/api/auth/me`, conversation list, `/api/chat/stream`, post-restart `/api/auth/me`, conversation list, and message history requests all reached the local API with HTTP 200. The restart did not issue a second login.
- Server prerequisite: `make server-db-smoke` passed all three real-database scenarios. The Chat/SSE case created its default model through the public API, consumed meta/think/token/done events, read the generated history through the public API, and directly confirmed the otherwise invisible two-row PostgreSQL invariant without repository-seeding messages.
- Automation: package-owned Maestro 2.6.1 flow, `1/1` passed in 48 seconds. Runtime-only credentials are expanded into the flow, and generated Maestro command JSON is removed on exit.
- Evidence: `output/ios-simulator/runs/20260719-chat-persistence-r5/maestro-junit.xml` and `evidence/screenshots/01-chat-stream-complete.png` through `03-chat-listed-in-drawer.png`; supporting API/provider logs are under `output/playwright/runs/20260719-chat-persistence-app/logs/`.
- Harness diagnosis before the passing run: delayed Expo development-client onboarding, repeated fixture model names, loopback-only App API binding, and a redundant development-client link after relaunch were corrected in setup/driver code. The product chat assertions passed on the first run that reached them; the final retry corrected only the relaunch driver.
- Cleanup: Metro, the local API, and the deterministic provider were stopped; run-owned containers, network, and volumes were removed by the root lifecycle.

#### 2026-07-19 profile and password lifecycle

- Environment: run-owned OrbStack PostgreSQL, Redis, and Elasticsearch; real migration and Server database prerequisite; local API at `http://192.168.2.142:58000`; Expo development client connected to run-owned Metro at port 8081.
- Simulator: iPhone Air, iOS 26.3, UDID `09864DA9-634C-4FA0-8900-4436A372F656`, portrait; bundle ID `io.github.chenyangzz.cove.mobile`.
- Covered flow: existing synthetic fixture login → profile nickname/email update → wrong-current-password rejection → successful password rotation → logout → old-password login rejection → new-password login and profile hydration.
- App-visible assertions: updated nickname/email persisted, `原密码错误` appeared without closing the password sheet, `邮箱或密码错误` appeared for the rotated-out password, and the updated profile rendered after login with the replacement password.
- Network assertions: Simulator-originated requests reached the local API; profile update returned 200, wrong password change returned 400, correct password change returned 200, old-password login returned 401, replacement-password login and `/api/auth/me` returned 200.
- Automation: package-owned Maestro 2.6.1 flow, `1/1` passed in 107 seconds. The flow uses stable App `testID` selectors and runtime-only synthetic credentials; generated Maestro command JSON is removed on exit because it expands injected values.
- Evidence: `output/ios-simulator/runs/20260718-profile-password/maestro-junit.xml`, `evidence/screenshots/01-profile-before-edit.png` through `06-new-password-login-success.png`, and sanitized `server-app-requests.log`.
- Harness diagnosis before the passing run: Expo development-client launcher/onboarding handling, iOS keyboard dismissal, cursor-position clearing for existing text, and nested alert accessibility were corrected in the project driver. These were harness/setup failures; the passing product flow was not retried after a product assertion failure.
- Cleanup: Metro and the local API were stopped; the `cove-e2e` containers, network, and volumes were removed through `make e2e-down`.

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
| Message history and ownership | API Chat/SSE via deterministic provider/Redis → API pagination/ownership → PostgreSQL | `make server-db-smoke` | Covered | Add generated partial/error message persistence and large-history pagination. |
| Profile and password changes | API profile/password/login/me → PostgreSQL | `make server-db-smoke` | Covered | Add avatar/object-storage coverage and explicit refresh-token revocation if password changes adopt that policy. |
| Model, agent, persona, MCP, skill, and tool configuration | API → PostgreSQL | — | Missing | Add representative CRUD and cross-user isolation flows. |
| Knowledge bases and document metadata | API/worker → PostgreSQL/local object storage | `make server-db-smoke` | Covered locally | Add cross-user document and knowledge-base ownership scenarios. |
| RAG indexing and retrieval | API/worker → PostgreSQL/Redis/Elasticsearch | `make server-db-smoke` | Covered locally | Add controlled embedding failures, retry/idempotency, and multi-document ranking scenarios. |
| Worker and scheduler tasks | API → Redis/asynq → worker → persistence | `make server-db-smoke` | Partially covered | Document parsing terminal success is covered; add retry/idempotency, scheduler, and other task types. |
| Gateway delivery | Gateway → queue/outbox → provider boundary | — | Missing | Add deterministic local provider and duplicate-event scenarios. |
| Expo App cross-boundary acceptance | `ios-simulator` skill/Maestro → Expo App → API → persistence | `make app-mobile-e2e-profile-password`, `make app-mobile-e2e-chat-persistence` | Partially covered | Continue with navigation/native lifecycle and the remaining auth/chat failure scenarios. |

## App acceptance backfill order

1. Expo authentication and SecureStore session lifecycle in iOS Simulator.
2. Expo profile/password flow against the local database environment.
3. Expo persistent chat and deterministic Chat/SSE creation.
4. Expo navigation, keyboard, sheet, and native-module lifecycle.
5. Expo native document selection, upload, worker parsing, and terminal chunk display.

Each App item must follow the `ios-simulator` skill and record an explicit Simulator UDID, the App-visible assertions, screenshot or recording paths, relevant sanitized logs, the first failing layer when applicable, and cleanup status. Skill-driven Computer Use is interactive evidence; mark deterministic automation separately only after a native regression framework owns the scenario.

## Server persistence backfill order

1. Representative configuration CRUD with cross-user isolation.
2. Deterministic Chat/SSE message creation using a local provider stub.
3. Knowledge upload and worker completion.
4. RAG retrieval, retries, and gateway delivery.

Each new row must use run-unique synthetic data, public assertions as the primary proof, direct database assertions only for otherwise invisible invariants, and scoped cleanup through the shared E2E lifecycle.
