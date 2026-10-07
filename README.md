# gh-aw-plus

A minimal fork of [GitHub Agentic Workflows (`github/gh-aw`)](https://github.com/github/gh-aw) with explicit Claude Code OAuth authentication.

Write an agent workflow in Markdown, compile it into GitHub Actions, and run the official Claude Code CLI using a `CLAUDE_CODE_OAUTH_TOKEN` GitHub secret. No `ANTHROPIC_API_KEY` secret is required in OAuth mode. Upstream API-key authentication, Anthropic WIF, default Claude behavior, and other engines remain supported.

This is an independent fork, not an official GitHub or Anthropic product. The target is a small, disposable extension that can disappear when upstream provides equivalent support. Updates follow published stable upstream releases; the selected release and commit are recorded in [FORK.md](FORK.md).

## Quick start

### Build and install the fork

The compiler is still named `gh-aw`, so existing `gh aw` commands work. This fork currently uses source builds rather than publishing its own release binaries.

Install GitHub CLI, the Go version specified in `go.mod`, and Node.js 24 or newer. Then:

```bash
git clone https://github.com/emmekappa/gh-aw-plus.git gh-aw
cd gh-aw
make deps
make build
./gh-aw --version
```

The checkout directory is named `gh-aw` so GitHub CLI discovers the extension as `aw`. To use it as the `gh aw` extension, remove an existing upstream installation if present, then install this local checkout:

```bash
gh extension remove aw   # Only if already installed
gh extension install .
gh aw --version
```

For Actions compilation, use the fork binary. Installing or upgrading `github/gh-aw` restores the upstream compiler, which does not include this extension. The upstream runtime setup action and pinned firewall remain unchanged; authentication is encoded by this compiler in the generated lock file.

### Configure Claude OAuth

Generate a token yourself using the official CLI:

```bash
claude setup-token
gh secret set CLAUDE_CODE_OAUTH_TOKEN --repo OWNER/REPOSITORY
```

Paste the token into the GitHub CLI's secret prompt. Keep it out of workflow source, command arguments, logs, and commits.

Create `.github/workflows/claude-oauth.md`:

```markdown
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
Summarize this repository's purpose and structure.
```

Compile and commit both the source and generated lock file:

```bash
gh aw compile claude-oauth
```

Run the workflow manually from GitHub Actions. The example has no schedule. Setting the secret alone does not enable OAuth: `auth.type: oauth` is required.

## Authentication and security

| Configuration | Credential | Behavior |
| --- | --- | --- |
| `engine: claude` | `ANTHROPIC_API_KEY` | Unchanged upstream default |
| `engine: { id: claude, auth: { type: oauth } }` | `CLAUDE_CODE_OAUTH_TOKEN` | Explicit subscription OAuth |
| Claude `auth.type: github-oidc`, `provider: anthropic` | Existing Anthropic WIF configuration | Unchanged keyless authentication |
| Other engines | Existing provider credentials | Unchanged; Claude OAuth mode is rejected |

OAuth uses the existing AWF API proxy. The actual token stays on the runner/proxy side and is excluded from the agent environment. Claude receives AWF's non-secret compatibility credential; the proxy injects bearer authentication. An internal host variable named `ANTHROPIC_API_KEY` transports the bearer credential to AWF's existing static credential interface. It references **only** `secrets.CLAUDE_CODE_OAUTH_TOKEN`; it does not require an API-key secret.

The network allowlist, sandbox, permission controls, MCP isolation, and log redaction remain upstream controls. This fork does not remove `--exclude-env`, bypass the firewall, or patch generated YAML. OAuth applies only to the built-in Claude engine using Anthropic inference.

See the [OAuth guide](docs/src/content/docs/guides/claude-oauth.md) for restrictions, environment-secret behavior, troubleshooting, and optional runtime tests. See the [research](docs/claude-oauth-research.md) for the removal history and the proof-of-concept analysis. Anthropic documents subscription tokens for its [Claude Code GitHub Action](https://code.claude.com/docs/en/github-actions); that is not a blanket approval of this independent integration or a guarantee about subscription limits.

## Test and develop

```bash
make deps
make build
make fmt
make fmt-check
make lint BASE_REF=v0.89.21 # Use the upstream tag recorded in FORK.md
go test ./pkg/workflow ./pkg/cli -run 'TestClaudeOAuth|TestWorkflowSecretsClaudeOAuth' -count=1
make test-unit-all
make test-js
make test-wasm-golden
make test-scripts
make test-integration-workflow
make test-integration-compile
make build-wasm
```

Use modern Bash and a filesystem with case-sensitive paths for the upstream test suite. Read the [validation record](docs/fork-validation.md) for the tested revision, results, and known upstream/environment failures. The OAuth tests use synthetic secrets. Real inference is separately opt-in and may consume subscription usage.

## Update from upstream manually

[FORK.md](FORK.md) defines the functional invariant, synchronized upstream commit, remaining delta, and removal plan. The canonical update procedure is [skills/sync-upstream/SKILL.md](skills/sync-upstream/SKILL.md).

In Codex, Claude Code, or GitHub Copilot coding agent, open this repository and give the agent this instruction:

```text
Read FORK.md and skills/sync-upstream/SKILL.md, then use the sync-upstream
skill to update this fork to the latest published stable github/gh-aw release.
Exclude prereleases and do not follow main. Preserve the OAuth functional
invariant and upstream security boundaries. Run the required checks,
commit the synchronization, and report the remaining fork delta.
```

This works without agent-specific installation. For native skill discovery, optionally link the canonical skill directory into the agent's project skill directory:

```bash
# Codex: then invoke $sync-upstream
mkdir -p .agents/skills
ln -s ../../skills/sync-upstream .agents/skills/sync-upstream

# Claude Code: then invoke /sync-upstream
mkdir -p .claude/skills
ln -s ../../skills/sync-upstream .claude/skills/sync-upstream
```

GitHub Copilot can load the repository skill entry in `.github/skills/sync-upstream/`. No synchronization workflow, timer, or scheduled agent is implemented.

## Upstream compatibility and contributing

Keep changes focused on OAuth, its tests, and fork maintenance. The Go module path and upstream runtime pins remain unchanged. Consult upstream [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) for engineering conventions; [FORK.md](FORK.md) is authoritative about the fork's purpose.

This implementation could form an upstream proposal: it is explicit, preserves default auth, and uses current engine/auth and AWF interfaces. Maintainer agreement and a real authenticated sandbox run are still needed before treating it as upstream-ready. No upstream PR has been submitted.

Upstream documentation: [GitHub Agentic Workflows](https://github.github.com/gh-aw/). Its published site describes upstream behavior; this fork's OAuth guide describes the extension.

## License

[MIT](LICENSE), inherited from upstream. Upstream attribution and licensing are retained.
