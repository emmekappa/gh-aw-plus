package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func isClaudeOAuth(data *WorkflowData) bool {
	return data != nil && data.EngineConfig != nil && data.EngineConfig.Auth != nil && data.EngineConfig.Auth.Type == "oauth"
}

func (e *ClaudeEngine) claudeAuthSecretNames(data *WorkflowData) []string {
	if isClaudeOAuth(data) {
		return []string{constants.ClaudeCodeOAuthToken}
	}
	return llmProviderSecretNames(e.ResolveLLMProvider(data))
}

func (e *ClaudeEngine) claudeAuthExcludedEnvNames(data *WorkflowData) []string {
	names := llmProviderSecretNames(e.ResolveLLMProvider(data))
	if isClaudeOAuth(data) {
		// Both the native token and AWF's host-only credential transport must
		// stay outside the agent, including when inherited through --env-all.
		names = append(names, constants.ClaudeCodeOAuthToken, "ANTHROPIC_AUTH_TOKEN")
	}
	return names
}

func applyClaudeOAuthEnv(env map[string]string, data *WorkflowData) {
	if !isClaudeOAuth(data) {
		return
	}
	delete(env, constants.AnthropicAPIKey)
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, constants.ClaudeCodeOAuthToken)
	token := "${{ secrets." + constants.ClaudeCodeOAuthToken + " }}"
	if isFirewallEnabled(data) {
		// AWF's static Anthropic credential slot supports a configurable header.
		// This is a bearer credential for the proxy, not an API-key requirement.
		// AWF strips it from the agent and supplies its own non-secret CLI token.
		env[constants.AnthropicAPIKey] = "Bearer " + token
		env["AWF_ANTHROPIC_AUTH_HEADER"] = "Authorization"
		// The CLI sees AWF's compatibility token rather than the real OAuth
		// token, so explicitly request Anthropic's OAuth API beta.
		if betas := env["ANTHROPIC_BETAS"]; betas != "" {
			env["ANTHROPIC_BETAS"] = betas + ","
		}
		env["ANTHROPIC_BETAS"] += "oauth-2025-04-20"
		env["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"] = "0"
	} else {
		env[constants.ClaudeCodeOAuthToken] = token
	}
}

// validateClaudeOAuthConfig keeps subscription OAuth separate from WIF and
// provider overrides. Existing authentication configurations are unaffected.
func validateClaudeOAuthConfig(config *EngineConfig, engine CodingAgentEngine, sandbox *SandboxConfig) error {
	if config == nil || config.Auth == nil || config.Auth.Type != "oauth" {
		return nil
	}
	if engine.GetID() != "claude" || config.IsInlineDefinition || resolveEngineLLMProvider(&WorkflowData{EngineConfig: config}, LLMProviderAnthropic) != LLMProviderAnthropic {
		return errors.New("engine.auth.type 'oauth' requires the built-in Claude engine with the Anthropic provider.\n\nExample:\nengine:\n  id: claude\n  auth:\n    type: oauth")
	}
	auth := *config.Auth
	auth.Type = ""
	if auth.Provider == "anthropic" {
		auth.Provider = ""
	}
	if auth != (EngineAuthConfig{}) {
		return errors.New("claude OAuth auth cannot include WIF fields or another auth provider. Use engine.auth.type: oauth with CLAUDE_CODE_OAUTH_TOKEN")
	}
	for key := range config.Env {
		if key == constants.AnthropicAPIKey || key == constants.ClaudeCodeOAuthToken || key == "ANTHROPIC_AUTH_TOKEN" || key == "AWF_ANTHROPIC_AUTH_HEADER" || strings.HasPrefix(key, "AWF_AUTH_") {
			return fmt.Errorf("claude OAuth manages authentication environment variable %q. Remove its engine.env override and set the CLAUDE_CODE_OAUTH_TOKEN GitHub secret", key)
		}
	}
	data := &WorkflowData{SandboxConfig: sandbox}
	if header := extractAPITargetAuthHeader(data, "anthropic"); header != "" && !strings.EqualFold(header, "Authorization") {
		return errors.New("claude OAuth requires the Authorization auth header. Remove sandbox.agent.targets.anthropic.auth-header or set it to Authorization")
	}
	return nil
}
