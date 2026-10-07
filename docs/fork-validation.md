# Fork validation record

Date: 2026-10-07. Selected stable release: [`v0.89.21`](https://github.com/github/gh-aw/releases/tag/v0.89.21), upstream commit `c35393777e5604a63721d09512263b1383301d4f`.
Implementation commit: `c510d44584` (`feat(claude): support OAuth authentication`).
Test commit: `acb6a2eaed` (`test(claude): cover OAuth authentication`).

Checks ran against this implementation/test source tree, including documentation commit `60cf54335f`; the subsequent stable-release policy commit changes documentation only. The original main-based publication is preserved on `archive/main-based-oauth`, including its earlier validation record. Raw diagnostics remain local because they contain machine paths. No real OAuth token was read or used. No assertions were disabled or weakened.

## Environment

Native checks: macOS ARM64, case-insensitive filesystem, Go 1.27.1 (module minimum 1.26.8), Node 24.15.0, modern Bash and GNU sed. Go tests used canonical temporary paths and `NO_COLOR` was unset where appropriate.

Full Go comparison: Debian Bookworm Linux x64 containers under ARM-host emulation, case-sensitive paths, Go 1.27.0 SDK with native ARM compiler targeting Linux AMD64 (`CGO_ENABLED=0`), Node 24.21.0, Bash/GNU tools and shellcheck. Actual test binaries ran as Linux x64. This is a local emulated environment, not a claim that upstream CI or every native platform was tested. The clean release and fork shared identical tools and commands. Containers received repository source read-only and task-owned caches, without host credentials or a Docker socket. Docker-image integration checks skip in that container environment; native integration exercised the available Docker checks.

The synthetic runtime check used the stable release's unchanged pins: AWF `v0.28.23` and official Claude Code CLI `2.1.273`, installed in isolated temporary directories. It exercised the actual environment builder/proxy and CLI against a loopback mock inference service.

## Results

| Check | Result |
| --- | --- |
| Pure release full Go suite (`make test-unit-all` command) on Linux x64 | Passed: 10,802 top-level tests, 34,172 test/subtest passing entries |
| Fork full Go suite on matching Linux x64 | Passed: 10,810 top-level tests, 34,202 test/subtest passing entries; compiler, Claude API-key/WIF and other engines included |
| Full JavaScript selection on Linux x64 | Completed across 425 files: 14,277 passed, six failed, 30 skipped; all six failures reproduced on pure release |
| Fork-specific OAuth/secret tests on native macOS | Passed |
| `TestClaudeOAuthRuntime` with stable AWF/CLI pins | Passed: correct bearer and OAuth beta headers, mock inference, excluded raw credential environment and agent shell isolation |
| `TestClaudeOAuthLiveIntegration` | Skipped; requires explicit opt-in and securely supplied real token |
| Native build, formatting and `make fmt-check` | Passed |
| `make lint BASE_REF=v0.89.21` | Passed; 80 upstream JavaScript warnings, no errors |
| `make test-integration-workflow` in Linux x64 | Passed, including command-group tests |
| `make test-integration-compile` on native macOS and Linux x64 | Passed |
| `make test-scripts` on Linux x64 | Passed: all Bash script tests |
| `make test-wasm` on Linux x64 | Build passed; binary golden harness: two passed, zero failed, two skipped because compilation was unavailable for those fixtures |
| `make test-security` and `make test-wasm-golden` | Passed; golden target includes defaults for all five engines |
| `make check-workflow-drift` | Passed: 299 workflows compiled, all generated locks unchanged |
| Documentation build | Passed: 262 pages, all internal links valid, no loader errors; external gallery sources supplied from authentic `githubnext/agentics` commit `5d11aa2a05ce2c943c085acb7b12b583f83ed375` |
| Documentation remark unit tests | Passed: two tests |
| Canonical/Copilot skill format validation and `make check-skill-file-paths` | Passed |
| `BASE_REF=v0.89.21 make agent-report-progress` | Standard Go lint, build, schema freshness and sampled impacted tests passed; custom linters reported 15 existing findings, all reproduced in pure release code |

A stable release is not a guarantee of a universally green test suite.

## Baseline and environment limits

The full native pure-release Go suite has nine failing top-level CLI tests. They concern case-sensitive workflow discovery, shellcheck test assumptions, x64-only Copilot toolcache fixtures on ARM64, and `TestChiSquarePerfectBalance` (p=0.9961696194311833 rather than 1). All workflow tests pass. The full release suite passes in Linux x64, without modifying assertions.

One initial native integration run failed `TestCommandGroupAssignments` because several commands were absent from its shared root command. The identical isolated fork test passes, and the unchanged pure-release command package passes both individually and in five repeated full-package runs. This initial transient failure is retained as evidence; it was not reproduced on the pure release and is not presented as a proven baseline failure. No command-registration implementation or tests are changed by the fork.

The initial JavaScript run was interrupted by a disk-space error and Docker daemon shutdown after 368 file results. Task-owned artifacts/caches were cleaned up, Docker was restored, and the remaining 57 files were run with the same source, dependency versions and tooling. Aggregate counts above combine completed files only; they do not claim one uninterrupted run. All other Linux suites had already completed.

Six JavaScript failures observed in the Linux x64 full suite were all reproduced on the pure release in the same container/tooling, using a detached release checkout and the identical dependency tree. The baseline subset ran 577 tests: 570 passed and seven failed (one additional baseline-only failure). These concern existing failure-handler expectations, a large Git-history range, fatal signals/watchdog timing under emulation, and an unreadable-file assertion when running as root. No upstream JavaScript source, tests, or dependency pins are changed.

The custom-linter gate's 15 messages concern existing function lengths, discarded type assertions, and slice indexing in existing engine/compiler code. They were matched against a separate pure-release custom-linter run. No unrelated refactor or suppression was introduced to silence them; workflow drift was checked separately because this gate stops before it.

The synthetic runtime check does not exercise Docker iptables or authenticate with Anthropic. Live subscription inference and a full authenticated GitHub Actions sandbox acceptance run remain unperformed. No real credentials were needed for this migration.

## Reproduce and extend validation

Use the README and OAuth guide's commands. Select the latest published stable release through GitHub release metadata, resolve its exact tag commit, and compare pure upstream and fork under matching tools. Never interpret prereleases, `main`, or `target_commitish: main` as the stable base. Document platform failures and skips separately rather than hiding them.

For authenticated acceptance, manually dispatch a workflow compiled by this fork with an independently supplied `CLAUDE_CODE_OAUTH_TOKEN` secret. Verify successful inference and absence of the real token in shell commands and logs while preserving AWF restrictions. Inherited GitHub Actions remain disabled on this public fork, preventing copied upstream schedules from executing. No synchronization schedule was added.
