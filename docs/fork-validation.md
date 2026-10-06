# Fork validation record

Date: 2026-10-06. Upstream base: `3fd44f8dc17a5fcdc45f653e7826853456b1839e`.
Implementation commit: `aea8e12658` (`feat(claude): support OAuth authentication`).
Test commit: `21674357da` (`test(claude): cover OAuth authentication`).

Checks ran against the identical implementation/test source tree before these commits; the final documentation commit adds records and instructions only. Raw diagnostics are retained locally and are not published because they contain machine paths. No real OAuth token was read or used. No tests or assertions were disabled to obtain a passing result.

## Environment

macOS ARM64, case-insensitive filesystem, Go 1.27.1 (module minimum 1.26.8), Node 24.15.0. Modern Bash 5.3.20 and GNU sed were installed for Linux-oriented upstream shell tests. Go tests used a canonical temporary directory and `NO_COLOR` was unset where appropriate. AWF v0.28.31 and the official Claude Code CLI 2.1.288 were installed in isolated temporary directories for the synthetic runtime test, then removed.

Docker initially reported an ARM64 server, but became unavailable while preparing a Linux validation image. Linux/container validation could not be completed. A temporary disk-space failure interrupted one focused run; after cleanup, the focused checks and complete Go suite were rerun successfully to completion. This does not mean every upstream assertion passed.

## Results

| Check | Result |
| --- | --- |
| `make deps`, native `make build` | Passed |
| `make fmt`, `make fmt-check`, explicit Prettier/syntax check for the runtime test | Passed |
| `make lint BASE_REF=upstream/main` | Passed; 266 existing JavaScript warnings, no errors |
| Fork tests: `go test ./pkg/workflow ./pkg/cli -run 'TestClaudeOAuth\|TestWorkflowSecretsClaudeOAuth' -count=1` | Passed |
| `TestClaudeOAuthRuntime` using actual generated env/exclusion settings, pinned AWF and official CLI | Passed: bearer header, OAuth beta, proxy inference, raw-token isolation and agent shell check |
| `TestClaudeOAuthLiveIntegration` | Skipped; explicit opt-in and securely supplied real token required |
| `make test-unit-all` | Completed: 35,674 passing test/subtest entries; 10 failing top-level CLI tests (15 entries including subtests); all workflow/compiler tests passed |
| `make test-js` | Completed: 16,225 passed, 17 failed, 37 skipped across 500 files |
| Focused baseline JS comparison | 14 of the 17 failures reproduced on unmodified upstream; the other three did not recur in the isolated subset |
| JS affected subset with canonical temporary paths, modern Bash, GNU sed and unset `NO_COLOR` | 684 passed, six failed across 12 files; remaining failures also appeared on upstream |
| `make test-security` | Passed regression tests and fuzz seed corpus |
| `make test-wasm-golden` | Passed, including defaults for all five engines |
| `make test-integration-workflow`, `make test-integration-compile` | Passed: 2,990 passing test/subtest entries, including Claude and Gemini WIF |
| `make test-scripts` | Passed through workflow drift/purity checks, then stopped at a GNU-sed assumption in the threat-pin test. That test passed separately with GNU sed. Remaining scripts were run separately: usage-artifact and MCP-binary checks passed; temporary-directory ownership check had one failure on macOS |
| `make test-wasm` | WASM build passed; binary golden harness skipped all four fixtures because compilation was unavailable in that environment. This is not four passing golden tests |
| `make build-docs` / final `npm run build` | Passed: 272 pages, internal links validated. Initial runs logged missing external gallery sources despite exit 0; final run supplied authentic `githubnext/agentics` workflow sources at `5d11aa2a05ce2c943c085acb7b12b583f83ed375` through `AGENTICS_WORKFLOWS_DIR` and had no loader errors |
| Docs remark unit tests | Passed after installing documentation dependencies |
| Docs wizard model target | Failed: the upstream Makefile references a test file absent at this base |
| Skill-format validation for canonical skill and Copilot entry; `make check-skill-file-paths` | Passed |
| `BASE_REF=upstream/main make agent-report-progress` | Standard Go lint, build, schema freshness and impacted samples passed; custom linters reported eight pre-existing findings in an otherwise unchanged part of the touched compiler file |
| `make check-workflow-drift` (run separately because the custom-linter gate stopped before it) | Passed: 323 workflows compiled; all generated locks unchanged |
| Publication audit | Changed/new files checked for real credential patterns and personal machine paths; none found. GitHub noreply commit identity used; only `main` is pushed |

The synthetic runtime test verifies real environment-building and proxy code plus the official CLI, using a loopback mock service. It does not run Docker iptables or authenticate against Anthropic. The optional real CLI test is separate from full GitHub Actions sandbox acceptance. Those remain unperformed.

## Upstream and platform failures

The 10 failing Go tests were reproduced on a separate checkout of the exact unmodified upstream base with matching native tools. They concern case-sensitive workflow discovery on a case-insensitive volume, shellcheck test assumptions, x64-only Copilot toolcache fixtures on ARM64 (which fall through to a nonexistent synthetic release), and `TestChiSquarePerfectBalance`. The latter expects p=1 but obtains 0.9961696194311833; it is unrelated to authentication. The checkout was removed after recording evidence.

The JavaScript failures involve macOS canonical paths/symlinks, shell recovery commands, Git includeIf/safe-directory behavior, prompt error expectations, and existing safe-output tests. None of the upstream action JavaScript implementation or test files was modified. The baseline subset reproduces 14 failures; correcting the native test environment reduces the subset to six remaining failures. The three full-suite-only failures were in missing-file/output behavior and did not recur standalone; do not interpret the isolated run as proving the whole JavaScript suite green.

The temporary-directory shell check, missing wizard-model test, and GNU-sed assumption are in unchanged upstream scripts. The custom-linter findings are two function-length warnings and six unchecked-index findings already present in `compiler_orchestrator_engine.go`; the OAuth change adds only a validation call in a different function. No unrelated refactor or lint suppression was added to silence them.

## Reproduce and extend validation

Use the README commands and the OAuth guide's synthetic/live integration instructions. For a complete portability check, run the unmodified upstream suite and this fork on a Linux x64 runner with Bash/GNU tools and case-sensitive paths, and retain comparable results. Investigate remaining baseline defects separately rather than expanding this fork's delta.

For authenticated acceptance, manually dispatch a workflow compiled by this fork with an independently supplied `CLAUDE_CODE_OAUTH_TOKEN` secret. Verify successful inference and absence of the real token in agent shell commands and logs while preserving AWF restrictions. This publication deliberately leaves inherited GitHub Actions disabled, so no copied upstream schedules execute on the fork. It adds no synchronization schedule.
