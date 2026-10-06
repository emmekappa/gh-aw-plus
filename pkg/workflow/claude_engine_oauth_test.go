//go:build !integration

package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeOAuthAuthSelection(t *testing.T) {
	t.Parallel()
	engine := NewClaudeEngine()
	for _, tc := range []struct {
		name   string
		auth   *EngineAuthConfig
		secret string
	}{
		{name: "default API key", secret: "ANTHROPIC_API_KEY"},
		{name: "OAuth", auth: &EngineAuthConfig{Type: "oauth"}, secret: constants.ClaudeCodeOAuthToken},
		{name: "WIF", auth: &EngineAuthConfig{Type: "github-oidc", Provider: "anthropic"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := &WorkflowData{NetworkPermissions: &NetworkPermissions{Allowed: []string{"defaults"}, Firewall: &FirewallConfig{Enabled: true}}, EngineConfig: &EngineConfig{ID: "claude", Auth: tc.auth}}
			names := engine.GetRequiredSecretNames(data)
			validation := strings.Join(engine.GetSecretValidationStep(data), "\n")
			if tc.secret == "" {
				assert.Empty(t, names)
				assert.Empty(t, validation)
				return
			}
			assert.Equal(t, []string{tc.secret}, names)
			assert.Contains(t, validation, "${{ secrets."+tc.secret+" }}")
			if tc.auth != nil {
				assert.NotContains(t, validation, "ANTHROPIC_API_KEY")
			}
		})
	}
	assert.Contains(t, engine.GetSupportedEnvVarKeys(), constants.ClaudeCodeOAuthToken)
}

func compileClaudeOAuthTest(t *testing.T, frontmatter string) (string, *WorkflowData) {
	t.Helper()
	compiler := NewCompiler()
	path := filepath.Join(t.TempDir(), "oauth.md")
	data, err := compiler.ParseWorkflowString("---\non: workflow_dispatch\nnetwork: defaults\n"+frontmatter+"\n---\nReply with a short greeting.\n", path)
	require.NoError(t, err)
	output, err := compiler.CompileToYAML(data, path)
	require.NoError(t, err)
	return output, data
}

func TestClaudeOAuthCompiledWorkflow(t *testing.T) {
	t.Parallel()
	output, _ := compileClaudeOAuthTest(t, "engine:\n  id: claude\n  auth:\n    type: oauth")
	assert.NotContains(t, output, "secrets.ANTHROPIC_API_KEY")
	assert.Contains(t, output, "secrets.CLAUDE_CODE_OAUTH_TOKEN")
	assert.Contains(t, output, "--exclude-env CLAUDE_CODE_OAUTH_TOKEN")
	assert.Contains(t, output, "--exclude-env ANTHROPIC_API_KEY")
	assert.Contains(t, output, "--exclude-env ANTHROPIC_AUTH_TOKEN")
	assert.Contains(t, output, "--exclude-env ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	assert.Contains(t, output, "AWF_ANTHROPIC_AUTH_HEADER: Authorization")
	assert.NotContains(t, output, "AWF_AUTH_TYPE: oauth")
	assert.Contains(t, output, "oauth-2025-04-20")
	assert.Contains(t, output, "GH_AW_SECRET_NAMES:")

	// Parse the emitted YAML to check credential scope, not just text presence.
	var lock map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(output), &lock))
	jobs := lock["jobs"].(map[string]any)
	var executionEnv map[string]any
	for jobName, jobValue := range jobs {
		job := jobValue.(map[string]any)
		assert.NotContains(t, job["env"], constants.ClaudeCodeOAuthToken)
		for _, value := range job["steps"].([]any) {
			step := value.(map[string]any)
			env, _ := step["env"].(map[string]any)
			if step["id"] == "agentic_execution" {
				require.Equal(t, "agent", jobName)
				executionEnv = env
				assert.Equal(t, "Bearer ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}", env["ANTHROPIC_API_KEY"])
				assert.NotContains(t, env, constants.ClaudeCodeOAuthToken)
			} else if _, ok := env[constants.ClaudeCodeOAuthToken]; ok {
				assert.Equal(t, "validate-secret", step["id"], "raw OAuth is only needed for the activation check")
			}
		}
	}
	require.NotNil(t, executionEnv)
}

func TestClaudeOAuthEnvironment(t *testing.T) {
	t.Parallel()
	for _, sandbox := range []bool{true, false} {
		t.Run(strconv.FormatBool(sandbox), func(t *testing.T) {
			data := &WorkflowData{NetworkPermissions: &NetworkPermissions{Allowed: []string{"defaults"}, Firewall: &FirewallConfig{Enabled: true}}, EngineConfig: &EngineConfig{ID: "claude", Auth: &EngineAuthConfig{Type: "oauth"}, Env: map[string]string{"ANTHROPIC_BETAS": "existing-beta"}}, SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Disabled: !sandbox}}}
			env := NewClaudeEngine().buildClaudeCommandEnv(data)
			if sandbox {
				assert.Equal(t, "Bearer ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}", env["ANTHROPIC_API_KEY"])
				assert.Equal(t, "Authorization", env["AWF_ANTHROPIC_AUTH_HEADER"])
				assert.Equal(t, "existing-beta,oauth-2025-04-20", env["ANTHROPIC_BETAS"])
				assert.Equal(t, "0", env["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"])
				assert.NotContains(t, env, constants.ClaudeCodeOAuthToken)
			} else {
				assert.Equal(t, "${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}", env[constants.ClaudeCodeOAuthToken])
				assert.NotContains(t, env, "ANTHROPIC_API_KEY")
				assert.NotContains(t, env, "AWF_ANTHROPIC_AUTH_HEADER")
			}
			filtered := FilterEnvForSecrets(env, NewClaudeEngine().GetRequiredSecretNames(data))
			assert.Equal(t, env, filtered)
		})
	}
}

func TestClaudeOAuthMissingSecret(t *testing.T) {
	t.Parallel()
	script, err := filepath.Abs("../../actions/setup/sh/validate_multi_secret.sh")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, token string
		valid       bool
	}{
		{name: "missing"}, {name: "whitespace", token: "   "}, {name: "placeholder", token: "undefined"},
		{name: "present", token: "synthetic-oauth-secret-never-print", valid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", script, constants.ClaudeCodeOAuthToken, "Claude Code", "https://example.com/auth")
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "GITHUB_STEP_SUMMARY=" + filepath.Join(t.TempDir(), "summary"), "CLAUDE_CODE_OAUTH_TOKEN=" + tc.token, "ANTHROPIC_API_KEY=synthetic-api-key"}
			output, err := cmd.CombinedOutput()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, string(output), constants.ClaudeCodeOAuthToken)
			}
			assert.NotContains(t, string(output), "synthetic-oauth-secret-never-print")
			assert.NotContains(t, string(output), "synthetic-api-key")
		})
	}
}

func TestClaudeOAuthCompilerRejectsIncompatibleConfig(t *testing.T) {
	t.Parallel()
	cases := []string{
		"engine:\n  id: claude\n  provider: openai\n  auth:\n    type: oauth",
		"engine:\n  id: claude\n  auth:\n    type: oauth\n    audience: https://example.com",
		"engine:\n  id: claude\n  auth:\n    type: oauth\n  env:\n    ANTHROPIC_API_KEY: '${{ secrets.OTHER }}'",
		"engine:\n  id: claude\n  auth:\n    type: oauth\nsandbox:\n  agent:\n    targets:\n      anthropic:\n        auth-header: x-api-key",
	}
	for _, engine := range []string{"codex", "copilot", "gemini", "pi"} {
		cases = append(cases, "engine:\n  id: "+engine+"\n  auth:\n    type: oauth")
	}
	for _, fm := range cases {
		t.Run(fm, func(t *testing.T) {
			_, err := NewCompiler().ParseWorkflowString("---\non: workflow_dispatch\npermissions:\n  contents: read\nnetwork: defaults\n"+fm+"\n---\nHello.\n", filepath.Join(t.TempDir(), "invalid.md"))
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "oauth")
		})
	}
}

func TestClaudeOAuthDefaultEngineRegressions(t *testing.T) {
	t.Parallel()
	for _, engine := range []string{"claude", "codex", "copilot", "gemini", "pi"} {
		t.Run(engine, func(t *testing.T) {
			output, _ := compileClaudeOAuthTest(t, "engine: "+engine)
			assert.NotContains(t, output, "CLAUDE_CODE_OAUTH_TOKEN")
			assert.NotContains(t, output, "oauth-2025-04-20")
			if engine == "claude" {
				assert.Contains(t, output, "secrets.ANTHROPIC_API_KEY")
				assert.Contains(t, output, "--exclude-env ANTHROPIC_API_KEY")
			}
		})
	}
}

func TestClaudeOAuthWIFRegression(t *testing.T) {
	t.Parallel()
	output, _ := compileClaudeOAuthTest(t, "permissions:\n  contents: read\n  id-token: write\nengine:\n  id: claude\n  auth:\n    type: github-oidc\n    provider: anthropic\n    federation-rule-id: fdrl_test\n    organization-id: org_test\n    service-account-id: svac_test")
	assert.Contains(t, output, "AWF_AUTH_TYPE: github-oidc")
	assert.Contains(t, output, "AWF_AUTH_PROVIDER: anthropic")
	assert.Contains(t, output, "--exclude-env ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	assert.NotContains(t, output, "Validate ANTHROPIC_API_KEY secret")
	assert.NotContains(t, output, "CLAUDE_CODE_OAUTH_TOKEN")
}

// TestClaudeOAuthRuntime exercises the compiler output with the pinned AWF
// environment builder/proxy and the real Claude CLI, using only a synthetic token.
func TestClaudeOAuthRuntime(t *testing.T) {
	source, cli := os.Getenv("GH_AW_OAUTH_AWF_SOURCE"), os.Getenv("GH_AW_OAUTH_CLAUDE_CLI")
	if source == "" || cli == "" {
		t.Skip("set GH_AW_OAUTH_AWF_SOURCE and GH_AW_OAUTH_CLAUDE_CLI for the synthetic runtime test")
	}
	output, _ := compileClaudeOAuthTest(t, "engine:\n  id: claude\n  auth:\n    type: oauth")
	var lock map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(output), &lock))
	agent := lock["jobs"].(map[string]any)["agent"].(map[string]any)
	var step map[string]any
	for _, v := range agent["steps"].([]any) {
		candidate := v.(map[string]any)
		if candidate["id"] == "agentic_execution" {
			step = candidate
			break
		}
	}
	require.NotNil(t, step)
	payload, err := json.Marshal(map[string]any{"env": step["env"], "run": step["run"], "awfVersion": string(constants.DefaultFirewallVersion), "cliVersion": string(constants.DefaultClaudeCodeVersion)})
	require.NoError(t, err)
	script, err := filepath.Abs("../../scripts/test-claude-oauth-runtime.cjs")
	require.NoError(t, err)
	cmd := exec.Command("node", script, source, cli)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir(), "TMPDIR=" + t.TempDir()}
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "synthetic runtime experiment: %s", result)
	assert.Contains(t, string(result), "PASS: compiled OAuth -> AWF isolation -> Claude CLI -> bearer proxy")
}

// This optional test makes a real inference request and never prints CLI output.
func TestClaudeOAuthLiveIntegration(t *testing.T) {
	if os.Getenv("GH_AW_RUN_OAUTH_INTEGRATION") != "1" {
		t.Skip("real inference requires GH_AW_RUN_OAUTH_INTEGRATION=1")
	}
	token := os.Getenv(constants.ClaudeCodeOAuthToken)
	if token == "" {
		t.Skip("CLAUDE_CODE_OAUTH_TOKEN is unavailable")
	}
	cli := os.Getenv("GH_AW_OAUTH_CLAUDE_CLI")
	if cli == "" {
		t.Skip("GH_AW_OAUTH_CLAUDE_CLI is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	workspace := t.TempDir()
	cmd := exec.CommandContext(ctx, cli, "--bare", "--no-session-persistence", "--tools", "", "--max-turns", "1", "--output-format", "json", "--print", "Reply with exactly OAUTH_OK.")
	cmd.Dir = workspace
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + workspace, "CLAUDE_CONFIG_DIR=" + workspace, "DISABLE_TELEMETRY=1", "DISABLE_ERROR_REPORTING=1", "CLAUDE_CODE_OAUTH_TOKEN=" + token}
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("Claude OAuth inference failed; CLI output withheld to protect credentials")
	}
	var response struct {
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}
	if json.Unmarshal(result, &response) != nil || response.IsError || !strings.Contains(response.Result, "OAUTH_OK") {
		t.Fatal("Claude OAuth inference did not return the expected response; output withheld")
	}
}
