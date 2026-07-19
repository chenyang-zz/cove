# Incremental Git Workflow

Use Git as the durable record of feature development. A request to implement a feature authorizes automatic local commits for completed work units under this rule; it does not authorize publishing or integration actions.

## 1. Work Units

- Split feature development into the smallest independently understandable, testable, and reversible units that still leave the repository in a coherent state.
- A work unit must implement one clear behavior or supporting capability. Keep its implementation, tests, required generated artifacts, contract updates, migrations, and directly related documentation in the same commit when separating them would make the commit incomplete or broken.
- Examples include one server use case with its tests, one App interaction with its tests, one API contract increment with compatible consumers, or one isolated test-harness capability.
- Do not use arbitrary time intervals, file counts, token limits, or conversational turns as commit boundaries. Do not create checkpoint commits for code that is knowingly broken or unvalidated.

## 2. Completion Gate

A work unit is complete only when all of the following are true:

- Its acceptance behavior is implemented and no known required part of that unit is deferred.
- Applicable formatting, static checks, focused tests, and required real-database or E2E validation have passed. Use the closest fast checks for intermediate units and run the full requirement-level validation before final handoff.
- `git diff --check` passes and generated artifacts are synchronized when the unit changes generated contracts or code.
- GitNexus change detection has been run before the commit and the affected symbols and execution flows match the intended scope. High or critical unexpected impact blocks the commit and must be reported.
- Run-owned services, fixtures, and temporary credentials are not staged. Resources may remain active only when the next unit explicitly reuses the same isolated run and ownership remains clear.

If a gate fails, fix the unit or report the blocker. Do not commit merely to clear the working tree.

## 3. Automatic Commit Procedure

- Once a work unit passes its completion gate, commit it automatically without waiting for an additional user confirmation. The original feature-development request is standing authorization for these local commits.
- Review `git status`, the unstaged diff, and the staged diff before every commit. Stage only explicit files or hunks owned by the current work unit; never absorb pre-existing, user-owned, generated-by-another-task, or unrelated changes.
- If the current unit overlaps an existing uncommitted change and safe hunk-level separation is not possible, stop and request direction instead of committing either change.
- Follow the repository commit format documented in `CONTRIBUTING.md`: `<emoji> <type>(<scope>): <Chinese summary>`, with an optional Chinese body explaining motivation and material behavior.
- Record the resulting commit SHA, subject, validation performed, and remaining units in task progress. Continue the next unit from the clean committed state.
- Do not rewrite, amend, squash, or reorder an earlier unit commit unless the user explicitly requests history cleanup and the commits have not been published.

## 4. Commit Scope and Ordering

- Prefer one logical work unit per commit. Avoid both large catch-all commits and mechanical one-file-per-commit fragmentation.
- Preserve dependency order. For cross-package API work, update and validate the Server contract first, then compatible App consumers, then E2E acceptance. Each intermediate commit must remain buildable and must not knowingly break an existing consumer.
- A contract change and all required consumers may remain one atomic commit when no safe compatible intermediate state exists.
- Tests that prove a behavior belong with the behavior. Additional broad regression coverage or test infrastructure may be a later unit when it is independently useful and the behavior commit is already adequately protected.
- The final feature handoff reports the ordered commit range and maps each commit to its completed work unit.

## 5. Authority Boundaries

- Automatic authorization covers local `git add` and `git commit` for completed work units only.
- Never automatically push, force-push, merge, cherry-pick into another branch, rebase, tag, publish a release, delete a branch, or alter remote state. These actions require explicit user authorization.
- Never bypass hooks or validation with `--no-verify`. Never use destructive Git cleanup to make a unit appear clean.
- If the task ends with an incomplete unit, leave its changes uncommitted, preserve the worktree, and report the exact state and blocker.
