---
name: sync-upstream
description: Update gh-aw-plus from github/gh-aw while preserving explicit Claude OAuth authentication and upstream security, minimizing or eliminating the fork delta.
---

# Synchronize gh-aw-plus with upstream

Use when asked to synchronize this fork with upstream. Run manually; do not create schedules, workflows, timers, or automated synchronization infrastructure. Read repository agent instructions and `FORK.md` before any changes. Code, commit messages, and repository documentation stay in English.

## Functional invariant

> This fork must preserve the ability to run the Claude engine using `CLAUDE_CODE_OAUTH_TOKEN` without requiring `ANTHROPIC_API_KEY`, while preserving upstream gh-aw security boundaries and all existing authentication mechanisms.

Preserve behavior, follow current upstream architecture, minimize the fork delta, and preserve security. Today's implementation is disposable. Never keep obsolete APIs merely to preserve old code or make it compile.

## 1. Establish the starting state

- Read `FORK.md`, `README.md`, the OAuth guide and research linked there, and applicable `AGENTS.md`/developer instructions.
- Inspect `git status`, remotes, branch, fork-specific commits, and diff against the documented upstream base. Record the current tested head and previous upstream base. Preserve unrelated or parked work; use an isolated worktree when needed.
- Confirm the current behavioral tests and credential propagation. Do not assume file names or architecture remain unchanged.

## 2. Fetch and identify upstream

Ensure remote `upstream` points to `https://github.com/github/gh-aw.git`; correct a wrong URL without changing `origin`. Fetch upstream branches and tags explicitly (a remote may have a default-branch-only refspec):

```bash
git fetch --prune --tags upstream '+refs/heads/*:refs/remotes/upstream/*'
```

Determine the remote default branch using `git ls-remote --symref upstream HEAD`, not a hard-coded branch name. Fetch its head explicitly if necessary and record the exact latest commit.

On a case-insensitive filesystem, upstream branch names may collide. Use a case-sensitive checkout/ref store or explicitly fetch the default branch and relevant branches after reporting the collision; do not pretend all branches were fetched. Preserve tags. Determine the current fork base using `FORK.md`, ancestry/merge-base, and history, and inspect commits since that base. Query the latest published release when relevant; distinguish a release from the newest tag and branch head.

## 3. Inspect changes before integrating

Review upstream commit summaries and diffs touching Claude, engine configuration, authentication, secret discovery/validation, environment allowlists/propagation, compiler, CLI invocation, AWF versions, sandboxing, `--exclude-env`, generated workflows, and related tests. Follow moved files and new abstractions. Record which changes affect OAuth and its security assumptions. Consult linked PRs/issues when code alone does not explain a support decision.

## 4. Determine whether the fork is still needed

Search upstream for native Claude subscription OAuth support, including `CLAUDE_CODE_OAUTH_TOKEN`, auth configuration, required secrets, firewall exclusions, and runtime tests. Test whether upstream can perform the invariant; a string match is insufficient.

If upstream provides equivalent or better functionality, remove redundant fork code, migrate configuration and tests, update `FORK.md` and docs, and report that the original reason for the fork may be resolved. Prefer zero fork-specific implementation. Do not reapply obsolete patches blindly.

## 5. Integrate and adapt

Choose merge or rebase according to shared history. Prefer merging published history unless rebasing was authorized; keep the remaining delta and commits readable. Perform a clean integration normally. Understand both sides of conflicts; never resolve wholesale with `ours` or `theirs`.

If OAuth still needs a patch, adapt it to the **current** upstream authentication abstraction. For example, adopt a new generic provider system instead of retaining Claude-specific legacy plumbing. Keep defaults, API keys, Anthropic WIF, inline auth, Codex, Copilot, Gemini, and Pi unchanged. Do not patch generated YAML with text replacement.

## 6. Review security

Inspect actual compiled `.lock.yml` and runtime propagation. Only use GitHub secret expressions; never read, echo, serialize, commit, or copy a real token. Check secret requirements, validation scope, environment filtering, redaction, CLI flags, provider endpoints, proxy credential injection, and shell access.

Review `--exclude-env`, AWF/sandbox changes and version pins. The real token must stay out of arbitrary agent shell commands under the default sandbox. Current host-only bearer transport and OAuth beta handling are documented in the guide, but may need replacement if upstream changes. Never weaken network, privilege, MCP, or credential isolation to make OAuth work. If upstream security changes prevent a safe implementation, stop integration and report the exact incompatibility; do not bypass it.

## 7. Validate

Discover current upstream test commands from its instructions/Makefile; do not assume old targets remain valid. Run fork OAuth tests, Claude/auth/WIF and other-engine regression tests, compiler tests, the upstream test suite, lint, formatting checks, and build. At the current base, useful commands include:

```bash
make deps
make build
make fmt
make fmt-check
make lint BASE_REF=upstream/main
go test ./pkg/workflow ./pkg/cli -run 'TestClaudeOAuth|TestWorkflowSecretsClaudeOAuth' -count=1
make test-unit-all
make test-js
make test-wasm-golden
make test-scripts
make test-integration-workflow
make test-integration-compile
make build-wasm
BASE_REF=upstream/main make agent-report-progress
```

Use the actual default branch in `BASE_REF`. Run the synthetic AWF/official CLI experiment described in the OAuth guide when credential plumbing or runtime pins change. Real inference requires explicit opt-in and securely supplied credentials; it is optional and must not expose tokens. A skipped live test is not authenticated acceptance.

Fix failures from legitimate architecture changes. Never hide, disable, or weaken upstream tests. For suspected baseline/environment failures, reproduce them on the exact unmodified upstream commit with matching tools and report evidence separately. An unexplained or security-related failure prevents claiming a safe synchronization. If validation is incomplete, leave the integration reviewable and clearly report the unfinished checks.

## 8. Review delta and update canonical records

Compare the result to the new upstream commit. Remaining divergence must be entirely OAuth support, its tests, fork documentation, or this skill. Flag unrelated changes and remove obsolete fork code. Check the diff for credentials and personal machine paths.

Update `FORK.md` after a successful synchronization with previous/new upstream commit, relevant tag/release, date, fork-specific commits and files, architecture changes, OAuth support status, test evidence, and removal plan. Update the OAuth guide, README, research and validation record when behavior or evidence changes. Keep the invariant explicit even if the implementation disappears.

## 9. Commit and report

Create identifiable commits, for example `chore: sync upstream github/gh-aw`, with meaningful OAuth adaptations in a separate `refactor(claude): adapt OAuth support to upstream auth changes` commit. Avoid unrelated edits. Follow repository validation gates before committing. Publishing follows the user's authorization; the skill alone does not authorize pushing or messaging others.

Report previous/new upstream bases, relevant upstream changes, conflicts and resolution, OAuth adaptations, security implications, checks and results (including skips/baseline failures), remaining delta, and whether the fork is still needed. If synchronization cannot be completed safely, explain exactly why and the reviewable repository state rather than forcing a merge.
