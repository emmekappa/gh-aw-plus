---
title: Claude Code OAuth in gh-aw-plus
description: Configure explicit Claude subscription OAuth while retaining Agent Workflow Firewall credential isolation.
---

This fork adds `engine.auth.type: oauth` to the built-in Claude engine. The default `engine: claude` still uses upstream API-key authentication. Anthropic WIF continues to use `auth.type: github-oidc`.

## Configure the workflow

Run `claude setup-token` yourself, then store the resulting token using the interactive `gh secret set CLAUDE_CODE_OAUTH_TOKEN --repo OWNER/REPOSITORY` prompt. The coding agent does not need to see the token.

```aw
---
on: workflow_dispatch
permissions:
  contents: read
engine:
  id: claude
  auth:
    type: oauth
network: defaults
---
Summarize the repository.
```

Compile with this fork's `gh aw compile` binary. Commit the Markdown and generated `.lock.yml` together. Recompilation preserves OAuth structurally; no post-compilation patch is needed.

OAuth requires `CLAUDE_CODE_OAUTH_TOKEN`. It has no API-key fallback. If both secrets exist, OAuth uses only the OAuth secret. Workflows without `auth.type: oauth` retain upstream behavior.

OAuth is restricted to the built-in Claude engine with the Anthropic inference provider. WIF fields and authentication-related `engine.env` overrides cannot be combined with it. `sandbox.agent.targets.anthropic.auth-header`, when configured, must be `Authorization`. Existing network and custom endpoint controls still apply; only send credentials to endpoints you trust.

## Credential propagation

With the default sandbox, the execution step supplies `Bearer ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}` through AWF's host-only `ANTHROPIC_API_KEY` credential slot and selects `AWF_ANTHROPIC_AUTH_HEADER: Authorization`. This is an implementation detail of the existing AWF interface, not a requirement for an `ANTHROPIC_API_KEY` GitHub secret.

`--exclude-env` excludes `ANTHROPIC_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, and `ANTHROPIC_AUTH_TOKEN` from host environment forwarding. AWF then injects its own non-secret compatibility token and proxy URL for the official Claude CLI. `ANTHROPIC_BETAS` includes `oauth-2025-04-20`, preserving other configured betas. The CLI's experimental-beta disabling flag is cleared in this mode because OAuth requires that beta.

The real credential is available only to trusted runner/proxy code and the activation secret check. Agent shell commands inherit no real OAuth token. AWF still strips client authentication and injects upstream bearer authentication. Network restrictions, privilege controls, MCP boundaries, and existing redaction are unchanged.

If a workflow explicitly disables its sandbox using upstream's non-strict configuration, the CLI receives the native `CLAUDE_CODE_OAUTH_TOKEN` variable directly. That inherits upstream's unsandboxed credential exposure. OAuth does not disable the sandbox itself.

GitHub environment secrets follow upstream's activation behavior: the early secret check is skipped when a top-level `environment` is configured, because that job cannot read the environment's secrets. The execution job still needs a valid token. No credential values are read by the compiler or written into lock files; only GitHub secret expressions are generated.

## Troubleshooting

- Missing token: set the secret on the repository or selected GitHub environment. An API key does not satisfy OAuth mode.
- Authentication rejected: obtain a new token with `claude setup-token` and replace the secret. Confirm subscription access and usage limits.
- OAuth ignored after recompilation: confirm that the compiling binary is this fork and the source explicitly selects OAuth.
- Firewall denied a connection: inspect the denied destination and upstream network configuration; retain the sandbox and credential exclusions.
- Conflicting authentication configuration: remove the conflicting WIF fields, `engine.env` authentication override, or non-Authorization proxy header.

## Optional runtime checks

The normal Go tests validate generated workflows, secret checks, environment selection, WIF, and other engines without credentials.

For the synthetic runtime experiment, check out the AWF version pinned in `pkg/constants/version_constants.go`, run `npm ci && npm run build` there, and run `npm ci` in its `containers/api-proxy` directory. Install the official Claude CLI version pinned in the same Go file in an isolated directory. Then set the paths:

```bash
GH_AW_OAUTH_AWF_SOURCE=/path/to/pinned-awf-checkout \
GH_AW_OAUTH_CLAUDE_CLI=/path/to/pinned/claude \
go test ./pkg/workflow -run '^TestClaudeOAuthRuntime$' -count=1 -v
```

This experiment consumes actual compiled env/exclusion settings, runs AWF's real environment builder and HTTP proxy, executes an agent shell credential check, and runs the official Claude CLI against a loopback mock provider. It verifies bearer headers, OAuth beta propagation, and inference response handling. Its mock HTTP transport uses an ephemeral local port; it does not exercise a Docker sandbox's iptables or Anthropic's subscription service.

A separate optional test runs the official CLI with a real token in an isolated directory:

```bash
# Supply CLAUDE_CODE_OAUTH_TOKEN securely in the environment; never echo it.
GH_AW_RUN_OAUTH_INTEGRATION=1 \
GH_AW_OAUTH_CLAUDE_CLI=/path/to/claude \
go test ./pkg/workflow -run '^TestClaudeOAuthLiveIntegration$' -count=1
```

The test skips unless both opt-in and a token are present. It can consume subscription usage and withholds CLI output. It tests native CLI authentication, not the full GitHub Actions sandbox. Full authenticated workflow acceptance should additionally use the manually dispatched example with the fork-generated lock file.
