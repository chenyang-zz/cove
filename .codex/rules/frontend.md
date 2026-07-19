# Cove App and Frontend Rules

These rules apply to work under `packages/app/`. Read `packages/app/AGENTS.md` as the authoritative package instruction file and read `.codex/rules/architecture.md` before making changes.

## 1. Scope and Sources of Truth

- Cove currently has one active frontend product: the Expo App under `packages/app/mobile/`.
- Treat `packages/app/frontend/` and the Wails shell as auxiliary or legacy code. Change or validate them only when a task explicitly targets them or a shared server contract requires a compatibility audit; do not treat their browser tests as App acceptance.
- Use the Cove Figma file referenced by `packages/app/AGENTS.md` for visual UI work.
- Verify every added or changed API request against `packages/server/docs/openapi.json`. When streaming behavior is not fully described there, inspect the server implementation and its SSE event definitions.
- Audit both clients when an API or SSE contract changes, even if the requested screen exists in only one client.
- Do not hard-code deployment URLs. Use `VITE_API_BASE_URL` for React/Vite and `EXPO_PUBLIC_API_BASE_URL` for Expo mobile. Keep environment-specific values in uncommitted local environment files.

## 1.1 Mandatory App Design Workflow

- For every new App screen or flow, and every material visual redesign of an existing App surface, load and use the `imagegen-frontend-mobile` skill before writing UI implementation code. This is a mandatory design gate for Cove's Expo App.
- Use the skill for work that requires visual or interaction-design judgment, including navigation composition, information hierarchy, layout, typography, palette, imagery, icon direction, component appearance, responsive states, onboarding, authentication, chat, profile, settings, empty states, errors, sheets, and multi-screen flows.
- The skill generates design images only. Complete image generation as a distinct design work unit; do not ask it to write React Native, SwiftUI, Flutter, or HTML. Begin the implementation work unit only after the generated screen set is available as the design reference.
- Default to `iOS-native premium` because iOS is Cove's primary App platform. Use Android-native or cross-platform mode only when the requirement explicitly targets that platform or a shared design must be proven on both platforms.
- Generate enough standalone screens to cover the affected user journey and its material states. Do not replace a flow with one compressed collage, crop old renders as new screens, or accept unreadable text, unsafe system regions, inconsistent mockups, generic template styling, or screen-to-screen design drift.
- Keep the generated set consistent with Cove's product identity, current Figma source, existing design tokens, navigation model, and buildable Expo/React Native constraints. When a generated concept conflicts with an established product contract, resolve the conflict before implementation rather than silently changing behavior.
- Treat the generated phone frame as presentation context, not App UI to implement. Translate the screen content, safe areas, hierarchy, components, and states into the real native layout.
- Preserve the generated images or conversation references long enough for implementation review. Do not commit generated design assets unless the user explicitly requests that they become repository artifacts.
- After implementation, use the `ios-simulator` skill to compare the running App against the generated reference for hierarchy, spacing, typography, colors, safe areas, interaction states, and complete flow behavior. Report material intentional differences.
- The design gate is not required for behavior-only changes with no visual effect, test-only selector or accessibility metadata changes that preserve appearance, dependency/tooling updates, or exact implementation of an already approved supplied design. If such work introduces design judgment, the gate becomes required.

## 2. Dependency and Configuration Discipline

- Use the package manager and lockfile already owned by each client. Both current clients use pnpm; do not introduce npm or Yarn lockfiles.
- Run mobile commands from `packages/app/mobile/`. Use `pnpm exec expo install <package>` for Expo-managed dependencies so versions remain compatible with the installed SDK.
- Do not use `--legacy-peer-deps`, pin an older transitive dependency, or add a resolver override merely because it fixed another Expo version. Reproduce the conflict in Cove first and document why the workaround is necessary.
- Do not edit `node_modules`. Preserve the checked-in `expo-modules-jsi` pnpm patch until an upstream version fixes the Xcode issue and a clean Simulator build verifies that the patch can be removed.
- Keep static application identity and plugin declarations in `packages/app/mobile/app.json`. Keep environment-dependent or computed Expo configuration in `packages/app/mobile/app.config.ts`.
- Run Expo prebuild after changing config plugins or native configuration. Inspect generated changes before retaining them; do not hand-edit generated native output when `app.json`, `app.config.ts`, a config plugin, or a patch is the real source of truth.
- Cove mobile currently targets Expo SDK 57 and React Native 0.86. Treat SDK 54/Fabric reports as diagnostic leads, not as proof of current behavior.

## 3. React/Vite Client

- Keep application shell and routing under `frontend/src/app/`; keep feature behavior under `frontend/src/features/`; keep reusable API adapters under `frontend/src/shared/api/`.
- Keep the authentication session contract in the auth feature and preserve the single refresh-and-retry behavior after HTTP 401.
- Keep chat streaming transport and SSE parsing in the chat API layer. Do not parse streams independently inside presentation components.
- Do not edit generated Wails bindings as handwritten code. Change the Go source or generation path, then regenerate and verify consumers.
- Preserve browser accessibility: semantic elements, keyboard operation, visible focus, meaningful labels, and non-color-only state indicators are required for new or changed interactions.

## 4. Expo Mobile Architecture

- Treat `packages/app/mobile/src/app/` and Expo Router as the owner of routes, native stack membership, protected routes, headers, gestures, and push/pop behavior.
- Keep authenticated session state in `AuthProvider`, transport and persistence in `src/core/`, reusable UI in `src/components/`, and visual tokens in `src/theme/`.
- Use `router.push()` for forward navigation, `router.back()` for a normal pop, and `router.replace()` only when history must be removed, such as crossing an authentication boundary.
- A screen with its own in-page header must set the owning `Stack.Screen` to `headerShown: false`; never render the native header and a custom header together.
- Use a development build rather than Expo Go for flows that depend on native modules such as `react-native-enriched-markdown`.
- Do not add a Metro resolver that silently replaces native modules with mocks. If an optional native capability must degrade in Expo Go, detect it at runtime, emit an explicit warning, and keep hook-using code in a separate inner component so hook order remains stable.
- If a Metro customization becomes necessary, extend Expo's default configuration and add only the smallest reproduced compatibility rule. Do not copy SDK-specific `tslib`, `.mjs`, or package-export workarounds without verifying the failure on Cove's current SDK.

## 5. Accessibility and Stable Automation Selectors

- Add both accessible semantics and a stable `testID` when adding or materially changing:
  - interactive controls such as `Pressable`, `TouchableOpacity`, `Switch`, and `TextInput`
  - clickable native header content
  - list-item root controls
  - critical success, error, alert, toast, and sheet actions
- Keep `testID` values in English kebab-case and describe product intent rather than visual appearance:
  - actions: `<domain>-<entity>-<action>`
  - inputs: `<domain>-<field>-input`
  - switches: `<domain>-<field>-switch`
  - navigation: `nav-<action>`
- Give dynamic list items a stable identifier with an index fallback, for example `chat-item-${item.id ?? index}`. Never generate repeated `*-undefined` selectors.
- Do not add `testID` to decorative containers or non-interactive text solely to increase selector count.
- Keep `accessibilityRole`, `accessibilityLabel`, state, and hint accurate. A `testID` is not a replacement for accessibility semantics.

## 6. Fabric-Safe Interaction and Layout

- Keep virtualized-list cell element trees stable. Do not use `Pressable` children-as-a-function inside `FlatList`, `SectionList`, or scrolling cells. Use `style={({ pressed }) => ...}` for pressed feedback and keep children static.
- Keep complex button layout in an inner `View`: padding, row/column layout, background, border radius, and content alignment belong there. Let the interactive wrapper own interaction, accessibility, hit slop, and any required outer width.
- Do not rely on an intrinsically sized `Pressable` as a complex flex-row layout owner. If a list row must fill remaining width and the behavior is unstable on Fabric, use a normal layout container or `TouchableOpacity` after validating both platforms.
- Put scroll-content padding on `contentContainerStyle`, not on the `ScrollView` style.
- Do not place percentage width, `aspectRatio`, `flexWrap`, or `flex: 1` in an auto-sized hierarchy unless an ancestor supplies a definite size. Validate these layouts on both iOS and Android.
- Keep selected indicators mounted and switch them to a transparent color when possible; avoid adding/removing absolute indicators when that causes layout movement.

## 7. Native Headers and Navigation Races

- Use `Pressable` from `react-native-gesture-handler` for interactive content returned by native `headerLeft`, `headerRight`, or `headerTitle` callbacks. Body controls may use React Native `Pressable`.
- Keep the root `GestureHandlerRootView` intact when using gesture-handler controls.
- Avoid combining an async mutation, structural state/cache updates, and a same-frame navigation pop on Android. If the flow reproduces a Fabric mount race, defer only the pop to the next animation frame behind a shared Cove helper.
- Keep direct `router.back()` for pure Back or Cancel actions that do not first mutate screen structure. Do not add an unnecessary delayed pop everywhere.
- Do not import a foreign `safeBack` path from another project. Add and test a Cove-owned helper only when the behavior is needed here.

## 8. Styling and Android Typography

- Use the React Native `boxShadow` string for new or materially changed standard surface shadows. Do not introduce a combined `shadowColor`/`shadowOffset`/`shadowOpacity`/`shadowRadius` plus `elevation` implementation for the same visual effect.
- Keep legacy shadow properties only for a measured platform-specific or animated effect. Do not mass-rewrite unrelated existing styles while completing a focused task.
- For fixed-height Android `TextInput` controls at 36 px or below, set `paddingVertical: 0` and `includeFontPadding: false`; avoid a fixed `lineHeight` that can clip glyphs.
- For Android badges or labels with `fontSize` at or below 11, set `includeFontPadding: false` and keep `lineHeight` close to the font size. Verify Chinese and numeric glyphs, not only Latin placeholders.
- Do not assume an iOS-correct result is cross-platform proof. Check Android for clipping, flex collapse, touch handling, and Fabric mount stability when the changed component is shared.

## 9. Sheets, Keyboards, and Animation Lifecycle

- Prefer React Native `Modal` plus `Animated` for a simple project-owned sheet instead of adding a heavy bottom-sheet dependency. If a third-party sheet is justified, verify its current Expo SDK 57/Fabric compatibility before adoption.
- Give a sheet a bounded height when its content uses `flex: 1`, percentage widths, wrapping, or `aspectRatio`. For auto-height sheets, use intrinsic content sizing and avoid those dependent layout combinations.
- Cove currently has no keyboard-controller compatibility layer. Use the existing React Native keyboard primitives unless a task justifies introducing one.
- If `react-native-keyboard-controller` is introduced, mount one provider at the root and expose it through a Cove compatibility module. Do not mount providers per page or mix direct imports with compatibility imports.
- When a condition change would unmount a component before its Reanimated feedback finishes, complete the animation first and invoke the state transition with `runOnJS`. Do not delay ordinary non-animated state changes.

## 10. Testing and Simulator Validation

- Treat `packages/app/mobile/` as the frontend validation scope. Passing React/Vite or Wails checks does not mean the App passed.
- Server unit tests, direct API tests, `make server-db-smoke`, and the auxiliary Playwright browser smoke are prerequisites or regression evidence, not App validation. For user-visible work, exercise the behavior in the Expo App through iOS Simulator or report that validation as missing.
- The workspace root `Makefile` exposes thin App wrappers such as `make app-build`, `make app-frontend-test`, and `make app-mobile-typecheck`; the underlying Taskfile and package scripts remain authoritative.
- Mobile tests use Vitest, not Jest. Do not add Jest-only configuration or mocks unless the project deliberately adopts Jest.
- From `packages/app/mobile/`, run `pnpm lint`, `pnpm typecheck`, and `pnpm test` for mobile code changes. Run prebuild or a native build when config, plugins, native dependencies, or runtime behavior changes.
- From `packages/app/frontend/`, run the relevant Vitest tests and `pnpm build` for React/Vite changes.
- Treat iOS as the primary mobile platform. Every App scenario must begin by loading and following `.agents/skills/ios-simulator/SKILL.md`; use its repository-discovery, build, lifecycle, layered-diagnosis, interaction, and evidence workflow instead of inventing per-task Simulator commands.
- Use Computer Use for exploratory or one-off foreground interaction when the Simulator accessibility tree is responsive. Use the package-owned Maestro workspace under `packages/app/mobile/e2e/maestro/` for deterministic native regression flows, including profile/password lifecycle coverage. Both paths remain subject to the Simulator skill baseline and local real-database rules.
- Select one explicit Simulator UDID. Prefer a supported Cove deep link for route setup, but never use a deep link to bypass the behavior being tested.
- Re-read the current Simulator state after each transition and prefer accessibility semantics or stable `testID` values. Coordinates are a fallback only when the current accessibility tree is incomplete.
- Capture screenshots for pixel state and recordings for navigation, animation, keyboard, or lifecycle behavior. For network and authentication scenarios, retain the App-visible result plus sanitized logs. A successful build, host-side request, Server test, or final still image does not prove the complete App interaction.
- Report the selected device and UDID, iOS version, build/launch command, installed bundle ID, Metro state, credential-free API base URL, exact actions and assertions, evidence paths, first failing layer, and cleanup status.
- Also run focused Android validation when touching shared Fabric layout, small typography, list virtualization, native headers, or navigation timing.

## 11. Applying External Gotchas

- Treat external gotcha documents as candidate failure patterns. Before turning a workaround into Cove code, verify the dependency exists, the version matches, the failure reproduces, and the proposed path exists in this repository.
- Do not copy tracking label maps, keyboard compatibility modules, navigation helpers, or other paths from a different repository into rules as if Cove already owns them.
- Promote a workaround to a permanent rule only after Cove reproduces the failure or intentionally adopts the related dependency and architecture.
