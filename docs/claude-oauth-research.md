# Claude OAuth research

Investigated against `github/gh-aw` main at `3fd44f8dc17a5fcdc45f653e7826853456b1839e` on 2026-10-06, before implementation. Runtime pins: AWF `v0.28.31`, Claude Code `2.1.288`. References describe their state at investigation time.

## Current architecture

`pkg/workflow/claude_engine.go` owns Claude secret requirements, validation, command environment, CLI invocation, and firewall exclusions. `EngineConfig.Auth` uses `EngineAuthConfig` in `pkg/workflow/engine.go`; it already represents `engine.auth.type: github-oidc`, including Anthropic WIF. Imported configuration is resolved before post-engine validation in `compiler_orchestrator_engine.go`.

`AuthDefinition` in `engine_definition.go` instead describes inline `engine.provider.auth` strategies (`api-key`, `bearer`, `oauth-client-credentials`). OAuth client credentials means exchanging a client ID/secret at a token endpoint. It is not the appropriate model for a supplied Claude subscription token. The extension therefore adds an explicit mode to the existing engine-auth configuration and keeps Claude credential translation in the Claude engine.

`pkg/constants/engine_constants.go` records default engine secrets and `AlternativeSecrets`; Claude currently has no alternatives. `pkg/cli/workflow_secrets.go` discovers credentials for a parsed workflow. The compiler filters secret expressions with the engine's required names; environment-key allowlists and global secret-name discovery also matter. OAuth must be recognized without making it an automatic alternative to the default API key.

Generated lock files use GitHub secret expressions, not credential values. Claude executes inside AWF by default. `--exclude-env` prevents selected runner variables from being forwarded through `--env-all`. AWF's API proxy supplies a placeholder Claude credential inside the agent and injects the real credential on the proxy side. The existing CLI command, networking, sandbox, MCP configuration, and WIF path remain intact.

## History and reasons for removal

| Evidence | What it establishes |
| --- | --- |
| [PR #3069](https://github.com/github/gh-aw/pull/3069), merged 2025-11-03, commit `aed701406b5a06d54fdc2de9341437c1404a769c` | OAuth was previously added as an alternative credential. Despite fallback wording in the PR description, the merged implementation supplied separate API-key and OAuth environment variables and validated that at least one secret existed. |
| [PR #9589](https://github.com/github/gh-aw/pull/9589), commit `82bf09b43b` | Secret filtering retained both Claude credential variables in that earlier architecture. |
| [Issue #16498](https://github.com/github/gh-aw/issues/16498) | A user reported OAuth failures with the then-current firewall, particularly on self-hosted ARM runners. Their analysis described an API-key-oriented proxy stripping headers and preventing bearer authentication from reaching Anthropic. This is a historical report, not proof of the current pinned proxy's behavior. |
| [Maintainer response](https://github.com/github/gh-aw/issues/16498#issuecomment-3942783340) | A maintainer stated that the token was not supported. |
| [PR #16757](https://github.com/github/gh-aw/pull/16757), merged 2026-02-19, commit `f0c707ef279bc84a1ee62a5b98b9e56d262bbefa` | Explicitly removed OAuth support, the alternative-secret entry, OAuth tests/fixture, and regenerated workflows. Its requested outcome included no backward compatibility. [PR #16735](https://github.com/github/gh-aw/pull/16735) was a duplicate proposal, not the merged removal. |
| [Later maintainer explanation](https://github.com/github/gh-aw/issues/16498#issuecomment-3981218075) | One stated concern was the impression that Anthropic was restricting transfer of a token between tools. The comment asked whether legitimate automated authentication existed and said support should be revisited if so. This is a concern and invitation to investigate, not an authoritative interpretation of Anthropic's terms. |
| [PR #20473](https://github.com/github/gh-aw/pull/20473) | Introduced the separate inline `AuthDefinition` scaffolding. |
| [PR #35939](https://github.com/github/gh-aw/pull/35939), [schema fix #36005](https://github.com/github/gh-aw/pull/36005), [documentation #39241](https://github.com/github/gh-aw/pull/39241) | Established the current Anthropic GitHub OIDC/WIF architecture. |
| [PR #36693](https://github.com/github/gh-aw/pull/36693) | Proposed additional blocking of OAuth variables. It was not merged and must not be mistaken for current behavior. |

The merged removal demonstrates an intentional support decision. Public evidence supports both a historical technical problem and maintainer concerns about supported automation. It does **not** establish a single definitive technical or contractual reason for removal. The removal PR itself does not provide a detailed rationale answering the user's request for clarification.

Current [official Claude Code GitHub Actions documentation](https://code.claude.com/docs/en/github-actions) documents `claude setup-token`, a `CLAUDE_CODE_OAUTH_TOKEN` GitHub secret, and subscription OAuth for supported plans. It recommends API-key or WIF authentication for shared organization repositories. That strengthens the case for discussing explicit support upstream, but does not establish official approval for this fork, the AWF proxy adaptation, or unlimited subscription use.

## Proof of concept

Examined [verkyyi/github-agent-runner](https://github.com/verkyyi/github-agent-runner/tree/d68826583f62c5f6a97121538464a7eecfd68a8b), specifically [`skills/install-workflow/auth.md`](https://github.com/verkyyi/github-agent-runner/blob/d68826583f62c5f6a97121538464a7eecfd68a8b/skills/install-workflow/auth.md).

Its workaround replaces `ANTHROPIC_API_KEY` in compiled locks with `CLAUDE_CODE_OAUTH_TOKEN`, then restores API-key names in `--exclude-env` arguments. This makes validation and execution reference the OAuth secret while allowing the native OAuth variable into the agent. Recompilation overwrites the patch.

Blind replacement is incorrect because validation, GitHub secret references, proxy credential transport, and security exclusions are different roles. Replacing exclusions can remove the only path by which the workaround delivers the token; removing OAuth exclusions exposes the real token to arbitrary agent shell commands. Hard-coded replacement counts also depend on the generated workflow structure. The PoC demonstrates a credential propagation problem, not an architecture to preserve.

## Pinned firewall investigation and experiment

Inspected [AWF v0.28.31](https://github.com/github/gh-aw-firewall/tree/v0.28.31), including `src/services/agent-environment/environment-builder.ts`, `src/services/api-proxy-credential-env.ts`, `containers/api-proxy/providers/anthropic.js`, and `containers/api-proxy/server.js`.

The static Anthropic proxy reads its credential from `ANTHROPIC_API_KEY` and supports `AWF_ANTHROPIC_AUTH_HEADER`. With `Authorization`, the configured value must include `Bearer `. The agent credential builder instead provides `ANTHROPIC_AUTH_TOKEN=sk-ant-placeholder-key-for-credential-isolation` and the proxy URL. The real host credential is stripped. OAuth can therefore use the existing proxy without giving the token to the agent or requiring an API-key GitHub secret.

The fork compiles the host-only slot as `Bearer ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}` and excludes all real Claude credential variables. The official CLI sees the placeholder, so the OAuth beta must be selected explicitly. An experiment showed that generic `ANTHROPIC_CUSTOM_HEADERS` was insufficient: the CLI overwrote its beta header. The documented [`ANTHROPIC_BETAS` variable](https://code.claude.com/docs/en/env-vars) correctly adds `oauth-2025-04-20`; the upstream experimental-beta disabling flag must also be cleared for this mode.

`TestClaudeOAuthRuntime` consumes the generated YAML settings and runs the pinned AWF environment builder, actual API-proxy request handler, a shell credential check, and the official pinned Claude CLI against a mock provider. It passed with a synthetic token, verifying bearer authentication, beta propagation, no API-key header, no real token in the agent environment or request body, and successful inference response handling. It does not prove Anthropic acceptance or Docker iptables enforcement. No real user token was used. The optional live test and manual workflow are documented separately.

## Upstream proposal assessment

This is a plausible upstream proposal because the mode is explicit, existing behavior is preserved, and it uses current engine/auth and proxy abstractions without YAML rewriting or firewall changes. Before an upstream submission, discuss the intended support policy with maintainers and perform a real authenticated sandbox workflow run. The beta/placeholder adaptation needs review against future CLI and AWF releases. Prefer an upstream-native OAuth credential provider if introduced; the fork implementation is disposable.
