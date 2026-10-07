---
name: sync-upstream
description: Update gh-aw-plus to the latest stable github/gh-aw release while preserving explicit Claude OAuth authentication and upstream security, minimizing or eliminating the fork delta.
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

## 2. Select and fetch the latest stable release

The default target is the GitHub-designated latest **published stable release**, not `main`, the highest tag, or the newest prerelease. Query `repos/github/gh-aw/releases/latest` using GitHub CLI/API and verify `draft: false` and `prerelease: false`. Set `STABLE_TAG` to the verified release `tag_name`; record its URL, publication date, and metadata. If no stable release exists or metadata cannot be verified, stop; never fall back silently to `main` or a prerelease. Development targets require an explicit user override, recorded in `FORK.md` and the report.

Ensure remote `upstream` points to `https://github.com/github/gh-aw.git`; correct a wrong URL without changing `origin`. Fetch branches and tags explicitly (a remote may have a default-branch-only refspec):

```bash
git fetch --prune --tags upstream '+refs/heads/*:refs/remotes/upstream/*'
```

On a case-insensitive filesystem, branch names may collide. Use a case-sensitive checkout/ref store, or fetch the selected release tag and relevant branches explicitly after reporting the limitation; do not pretend all branches were fetched. Determine the default branch for diagnostics only. Its head is not the integration target.

Resolve the fetched tag to its exact commit with `git rev-parse "refs/tags/$STABLE_TAG^{commit}"`. Verify it matches the remote tag; stop if an existing tag unexpectedly moved. Do not use the release's `target_commitish: main` as a commit pin. Use this tag/commit for integration, diff review, baseline tests, and `BASE_REF`. Record the current fork base from `FORK.md` and ancestry/history, and inspect changes between that base and the selected release. If already synchronized, report that fact rather than integrating newer development commits.

## 3. Inspect changes before integrating

Review commit summaries and diffs through the selected release, touching Claude, engine configuration, authentication, secret discovery/validation, environment allowlists/propagation, compiler, CLI invocation, AWF versions, sandboxing, `--exclude-env`, generated workflows, and related tests. Follow moved files and abstractions in that release. Record which changes affect OAuth and its security assumptions. Consult linked PRs/issues when code alone does not explain a support decision.

## 4. Determine whether the fork is still needed

Search the selected upstream release for native Claude subscription OAuth support, including `CLAUDE_CODE_OAUTH_TOKEN`, auth configuration, required secrets, firewall exclusions, and runtime tests. Test whether upstream can perform the invariant; a string match is insufficient.

If upstream provides equivalent or better functionality, remove redundant fork code, migrate configuration and tests, update `FORK.md` and docs, and report that the original reason for the fork may be resolved. Prefer zero fork-specific implementation. Do not reapply obsolete patches blindly.

## 5. Integrate and adapt

A release may not contain commits already present in a fork previously based on `main`. Merging an older tag cannot remove those commits. For an explicitly authorized migration, preserve the published branch, rebuild a separate branch from the release tag, and reapply only the functional fork delta. Compare the result directly against the release. Replacing published history requires authorization and a lease against the observed remote head; do not overwrite concurrent changes. If the selected release is older or divergent without such authorization, stop and explain the required migration.

Choose merge or rebase according to shared history. Prefer merging published history unless rebasing was authorized; keep the remaining delta and commits readable. Perform a clean integration normally. Understand both sides of conflicts; never resolve wholesale with `ours` or `theirs`.

If OAuth still needs a patch, adapt it to the authentication abstraction in the **selected release**. For example, adopt a generic provider system introduced in that release instead of retaining Claude-specific legacy plumbing. Keep defaults, API keys, Anthropic WIF, inline auth, Codex, Copilot, Gemini, and Pi unchanged. Do not patch generated YAML with text replacement.

## 6. Review security

Inspect actual compiled `.lock.yml` and runtime propagation. Only use GitHub secret expressions; never read, echo, serialize, commit, or copy a real token. Check secret requirements, validation scope, environment filtering, redaction, CLI flags, provider endpoints, proxy credential injection, and shell access.

Review `--exclude-env`, AWF/sandbox changes and version pins. The real token must stay out of arbitrary agent shell commands under the default sandbox. Current host-only bearer transport and OAuth beta handling are documented in the guide, but may need replacement if upstream changes. Never weaken network, privilege, MCP, or credential isolation to make OAuth work. If upstream security changes prevent a safe implementation, stop integration and report the exact incompatibility; do not bypass it.

## 7. Validate

First validate the unmodified selected release in a clean checkout, preferably on Linux x64 with the same tools as upstream CI. Record its exact commit, environment, checks and results. A release label does not guarantee a green suite. Then validate the fork under matching conditions so baseline failures and OAuth regressions can be distinguished.

Discover the selected release's test commands from its instructions/Makefile; do not assume old targets remain valid. Run fork OAuth tests, Claude/auth/WIF and other-engine regression tests, compiler tests, the upstream test suite, lint, formatting checks, and build. At the current base, useful commands include:

```bash
make deps
make build
make fmt
make fmt-check
make lint BASE_REF="$STABLE_TAG"
go test ./pkg/workflow ./pkg/cli -run 'TestClaudeOAuth|TestWorkflowSecretsClaudeOAuth' -count=1
make test-unit-all
make test-js
make test-wasm-golden
make test-scripts
make test-integration-workflow
make test-integration-compile
make build-wasm
BASE_REF="$STABLE_TAG" make agent-report-progress
```

Use the selected release tag/commit in `BASE_REF`; never the default branch. Compare matching baseline/fork environments. Run the synthetic AWF/official CLI experiment described in the OAuth guide when credential plumbing or runtime pins change. Real inference requires explicit opt-in and securely supplied credentials; it is optional and must not expose tokens. A skipped live test is not authenticated acceptance.

Fix failures from legitimate architecture changes. Never hide, disable, or weaken upstream tests. For suspected baseline/environment failures, reproduce them on the exact unmodified upstream commit with matching tools and report evidence separately. An unexplained or security-related failure prevents claiming a safe synchronization. If validation is incomplete, leave the integration reviewable and clearly report the unfinished checks.

## 8. Review delta and update canonical records

Compare the result to the selected release commit. Remaining divergence must be entirely OAuth support, its tests, fork documentation, or this skill. Flag unrelated changes and remove obsolete fork code. Check the diff for credentials and personal machine paths.

Update `FORK.md` after a successful synchronization with previous/new upstream commit, selected stable tag and release URL, synchronization channel, date, fork-specific commits and files, architecture changes, OAuth support status, test evidence, and removal plan. Update the OAuth guide, README, research and validation record when behavior or evidence changes. Keep the invariant explicit even if the implementation disappears.

## 9. Commit and report

Create identifiable commits, for example `chore: sync upstream github/gh-aw release <tag>`, with meaningful OAuth adaptations in a separate `refactor(claude): adapt OAuth support to upstream auth changes` commit. Avoid unrelated edits. Follow repository validation gates before committing. Publishing follows the user's authorization; the skill alone does not authorize pushing or messaging others.

Report previous/new upstream bases and release tags, selected channel, relevant upstream changes, conflicts and resolution, OAuth adaptations, security implications, checks and results (including skips/baseline failures), remaining delta, and whether the fork is still needed. If synchronization cannot be completed safely, explain exactly why and the reviewable repository state rather than forcing a merge.
