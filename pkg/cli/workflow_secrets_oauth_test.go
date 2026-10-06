//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowSecretsClaudeOAuth(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "oauth.md")
	require.NoError(t, os.WriteFile(path, []byte("---\non: workflow_dispatch\nengine:\n  id: claude\n  auth:\n    type: oauth\n---\nHello.\n"), 0600))
	reqs := getSecretRequirementsForWorkflow(path)
	require.Len(t, reqs, 1)
	assert.Equal(t, constants.ClaudeCodeOAuthToken, reqs[0].Name)
	assert.True(t, reqs[0].IsEngineSecret)
	assert.False(t, reqs[0].Optional)
	assert.Empty(t, getMissingRequiredSecrets(reqs, map[string]struct{}{constants.ClaudeCodeOAuthToken: {}}))
	assert.Len(t, getMissingRequiredSecrets(reqs, map[string]struct{}{constants.AnthropicAPIKey: {}}), 1)
}
