# gh-aw-plus fork contract

Upstream: [github/gh-aw](https://github.com/github/gh-aw), remote `https://github.com/github/gh-aw.git`.
Fork: [emmekappa/gh-aw-plus](https://github.com/emmekappa/gh-aw-plus).

## Functional invariant

> This fork must preserve the ability to run the Claude engine using `CLAUDE_CODE_OAUTH_TOKEN` without requiring `ANTHROPIC_API_KEY`, while preserving upstream gh-aw security boundaries and all existing authentication mechanisms.

Preserve this behavior, follow current upstream architecture, minimize the fork delta, and preserve security. The implementation is disposable. Equivalent upstream support should eliminate the fork-specific implementation.

## Synchronized base and purpose

Initial upstream base: `3fd44f8dc17a5fcdc45f653e7826853456b1839e` on `main`, fetched and inspected on 2026-10-06. Nearest ancestor tag: `v0.91.2` (six commits behind the base). The GitHub latest-release endpoint returned `v0.89.21`; a published release, newest tag, and latest main are distinct. No upstream synchronization has subsequently been performed.

At this base upstream explicitly does not support Claude subscription OAuth. See [research](docs/claude-oauth-research.md) for merged removal history, issues, PoC analysis, current architecture, and upstream proposal assessment.

The sole runtime extension is explicit OAuth for the built-in Claude engine with Anthropic inference:

```yaml
engine:
  id: claude
  auth:
    type: oauth
```

Store a token obtained with `claude setup-token` as the GitHub secret `CLAUDE_CODE_OAUTH_TOKEN`. Compile using this fork. OAuth has no API-key fallback. Merely defining the token does not change default Claude authentication.

## Architecture and invariants

- Use `EngineConfig.Auth` / `EngineAuthConfig` for the explicit mode. Inline `AuthDefinition` OAuth client credentials is a different protocol and remains unchanged.
- ClaudeEngine owns required secrets, validation, credential environment, and firewall exclusions. The compiler validates unsupported combinations after imported configuration is resolved.
- Default Claude API keys, existing provider overrides, Anthropic GitHub OIDC/WIF, and other engines retain upstream behavior. Other engines cannot select Claude OAuth.
- Do not restore `AlternativeSecrets` fallback behavior or replace API-key names globally. Workflow-aware secret discovery selects the explicit mode.
- Generated YAML uses only GitHub secret expressions. Never patch `.lock.yml` using `sed` or embed credential values.
- Preserve upstream firewall, sandbox, network, privilege, MCP, and redaction controls. Runtime pins and CLI invocation remain upstream's.
- No scheduled synchronization or fork release infrastructure is added.

With the default AWF sandbox, the host-only `ANTHROPIC_API_KEY` slot contains `Bearer ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}` and `AWF_ANTHROPIC_AUTH_HEADER` selects Authorization. This internal variable is the pinned AWF static credential interface, **not** a required API-key GitHub secret. Raw OAuth/API/auth-token variables remain excluded from agent forwarding. AWF supplies its own non-secret CLI credential and injects bearer auth in its proxy. OAuth requires explicit `ANTHROPIC_BETAS` propagation and clears the CLI experimental-beta disabling flag. See the [OAuth guide](docs/src/content/docs/guides/claude-oauth.md).

## Security and limits

The compiler never reads token values. The token is scoped to the activation check and trusted execution/proxy process; it is not a job-level credential and does not reach arbitrary agent shell commands under the default sandbox. Explicit unsandboxed execution retains upstream's broader credential exposure. Environment secrets retain upstream's skipped activation check and must be available to the execution job. Authentication overrides incompatible with OAuth are rejected.

Existing upstream proxy logging/redaction remains unchanged; this fork adds no token printing. Trusted runner/proxy code necessarily handles the credential. Custom inference endpoints must be trusted. Do not weaken exclusions or sandbox controls to bypass a failed authentication check.

The synthetic runtime experiment verifies the actual pinned proxy and official CLI, including shell isolation and provider headers. It does not verify Docker iptables or acceptance of a real subscription token. Real inference is optional and opt-in; no real credentials were used for initial validation. Official action support is not a blanket endorsement of this integration or subscription usage policy.

## Files in the current fork delta

- `pkg/constants/engine_constants.go`
- `pkg/parser/schemas/main_workflow_schema.json`
- `pkg/workflow/engine.go`
- `pkg/workflow/claude_engine.go`
- `pkg/workflow/claude_engine_oauth.go`
- `pkg/workflow/compiler_orchestrator_engine.go`
- `pkg/workflow/claude_engine_oauth_test.go`
- `pkg/cli/workflow_secrets.go`
- `pkg/cli/workflow_secrets_oauth_test.go`
- `scripts/test-claude-oauth-runtime.cjs`
- `README.md`
- `FORK.md`
- `docs/claude-oauth-research.md`
- `docs/fork-validation.md`
- `docs/src/content/docs/guides/claude-oauth.md`
- `docs/src/content/docs/engines/claude.md`
- `docs/src/content/docs/reference/auth.mdx`
- `docs/src/content/docs/reference/faq.md`
- `docs/src/content/docs/setup/cli.md`
- `docs/src/content/docs/setup/quick-start.mdx`
- `skills/sync-upstream/SKILL.md`
- `.github/skills/sync-upstream/SKILL.md`

No generated upstream workflow, action/runtime code, dependency version, module path, or other-engine implementation is modified.

## Fork-specific commits

The initial extension is organized as:

1. `aea8e12658` — `feat(claude): support OAuth authentication` — auth/schema/engine/secret discovery.
2. `21674357da` — `test(claude): cover OAuth authentication` — compilation, validation, environment, regressions, and optional runtime/live checks.
3. `docs(fork): document Claude OAuth extension` — this contract, README, guides, research, validation, and manual synchronization skill.

Use `git log --oneline 3fd44f8dc17a5fcdc45f653e7826853456b1839e..HEAD` for exact commit IDs, including the documentation commit itself. Update this section after each synchronization rather than retaining obsolete commit identifiers.

## Tests protecting the capability

`pkg/workflow/claude_engine_oauth_test.go` protects explicit/default/WIF secret selection, generated lock credential scope, no API-key-secret requirement, missing/invalid-token validation, direct/proxied environment propagation, exclusion of raw credentials, OAuth beta preservation, incompatible config rejection, and all other engine defaults. `pkg/cli/workflow_secrets_oauth_test.go` protects workflow-aware secret requirements and prevents an API key from satisfying OAuth.

`TestClaudeOAuthRuntime` plus `scripts/test-claude-oauth-runtime.cjs` checks generated configuration using the pinned real AWF environment builder/proxy and official Claude CLI with a synthetic token. `TestClaudeOAuthLiveIntegration` is an opt-in native CLI check. Neither replaces a full authenticated GitHub Actions sandbox acceptance run. Current commands and validation evidence are in [the guide](docs/src/content/docs/guides/claude-oauth.md) and [validation record](docs/fork-validation.md).

## Manual synchronization

Read and invoke [skills/sync-upstream/SKILL.md](skills/sync-upstream/SKILL.md). The `.github/skills/sync-upstream/SKILL.md` entry routes agents to that canonical procedure. [README.md](README.md) contains agent-neutral instructions and optional native skill discovery for Codex, Claude Code, and Copilot.

The procedure inspects changes before integration, checks whether upstream already satisfies the invariant, adapts to current abstractions, reviews credential isolation, runs upstream and fork tests, reviews the complete remaining diff, updates this contract, creates readable commits, and reports evidence. Published history should ordinarily be merged; rebase only with appropriate authorization. No automatic `ours`/`theirs` conflict resolution.

## Removal when upstream gains equivalent support

Verify equivalent upstream behavior with secret-requirement, compiled workflow, and runtime isolation tests. Migrate configuration to upstream's supported syntax, remove redundant OAuth engine/CLI/constants/schema plumbing, and adapt or remove overlapping tests. Remove the fork-specific runtime experiment if upstream supplies equivalent evidence. Update README, guides, this contract, and the skill to report that the original fork purpose is resolved. Retain only maintenance documentation that remains useful. The ideal outcome is zero fork-specific runtime code.
