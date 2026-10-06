// Optional synthetic experiment; invoked by TestClaudeOAuthRuntime.
// Dependencies live in a separate checkout of the compiler-pinned AWF release.
const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const path = require("node:path");
const { spawn, spawnSync } = require("node:child_process");

async function main() {
  const [source, cli] = process.argv.slice(2);
  const input = JSON.parse(fs.readFileSync(0, "utf8"));
  const version = JSON.parse(fs.readFileSync(path.join(source, "package.json")));
  assert.equal(`v${version.version}`, input.awfVersion, "use the pinned AWF release");
  assert.match(spawnSync(cli, ["--version"], { encoding: "utf8" }).stdout, new RegExp(input.cliVersion.replaceAll(".", "\\.")));
  const token = "sk-ant-oat01-synthetic-never-a-real-credential";
  const excluded = [...input.run.matchAll(/--exclude-env ([A-Z_][A-Z0-9_]*)/g)].map(match => match[1]);
  assert(excluded.includes("CLAUDE_CODE_OAUTH_TOKEN"));
  assert(excluded.includes("ANTHROPIC_API_KEY"));
  // Read the actual emitted env values, resolving only the OAuth secret locally.
  const hostEnv = {};
  for (const key of ["ANTHROPIC_API_KEY", "AWF_ANTHROPIC_AUTH_HEADER", "ANTHROPIC_BETAS"]) {
    hostEnv[key] = input.env[key].replace("${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}", token);
  }
  Object.assign(process.env, hostEnv, { CLAUDE_CODE_OAUTH_TOKEN: token });
  const config = {
    agentCommand: "claude",
    containerRuntime: "runc",
    envAll: true,
    excludeEnv: excluded,
    enableApiProxy: true,
    anthropicApiKey: hostEnv.ANTHROPIC_API_KEY,
    anthropicApiAuthHeader: hostEnv.AWF_ANTHROPIC_AUTH_HEADER,
    containerWorkDir: process.env.HOME,
  };
  const networkConfig = { squidIp: "127.0.0.1", proxyIp: "127.0.0.1", agentIp: "127.0.0.1", subnet: "127.0.0.0/24" };
  const { buildAgentEnvironment } = require(path.join(source, "dist/services/agent-environment/environment-builder.js"));
  const { buildAgentCredentialEnv } = require(path.join(source, "dist/services/api-proxy-credential-env.js"));
  const agentEnv = {
    ...buildAgentEnvironment({ config, networkConfig, dnsServers: ["127.0.0.1"] }),
    ...buildAgentCredentialEnv({ config, networkConfig }),
  };
  assert.equal(agentEnv.CLAUDE_CODE_OAUTH_TOKEN, undefined);
  assert.equal(agentEnv.ANTHROPIC_API_KEY, undefined);
  assert.equal(agentEnv.ANTHROPIC_AUTH_TOKEN, "sk-ant-placeholder-key-for-credential-isolation");
  assert(!Object.values(agentEnv).some(value => value.includes(token)));
  const shell = spawnSync("bash", ["-c", 'test -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" && test -z "${ANTHROPIC_API_KEY:-}"'], { env: agentEnv });
  assert.equal(shell.status, 0, "arbitrary agent shell commands cannot read the credential");

  const { createAnthropicAdapter } = require(path.join(source, "containers/api-proxy/providers/anthropic.js"));
  const { createProviderServer } = require(path.join(source, "containers/api-proxy/server.js"));
  const adapter = createAnthropicAdapter({ ...hostEnv, ANTHROPIC_API_TARGET: "http://127.0.0.1" });
  let requests = 0;
  const failures = [];
  // Mock provider; the real AWF request handler injects authentication.
  const backend = http.createServer((req, res) => {
    let body = "";
    req.on("data", chunk => {
      body += chunk;
    });
    req.on("end", () => {
      try {
        assert.equal(req.headers.authorization, `Bearer ${token}`);
        assert.equal(req.headers["x-api-key"], undefined);
        assert(!body.includes(token));
        if (!req.url.startsWith("/v1/messages")) {
          res.end("{}");
          return;
        }
        assert(req.headers["anthropic-beta"].split(",").includes("oauth-2025-04-20"), "OAuth beta missing: " + req.headers["anthropic-beta"]);
        if (req.url.startsWith("/v1/messages/count_tokens")) {
          res.setHeader("content-type", "application/json");
          res.end('{"input_tokens":10}');
          return;
        }
        const request = JSON.parse(body);
        assert.equal(request.stream, true);
        requests++;
        res.setHeader("content-type", "text/event-stream");
        const emit = (type, data) => res.write(`event: ${type}\ndata: ${JSON.stringify({ type, ...data })}\n\n`);
        emit("message_start", { message: { id: `msg_${requests}`, type: "message", role: "assistant", model: "claude-sonnet-4-6", content: [], stop_reason: null, stop_sequence: null, usage: { input_tokens: 10, output_tokens: 0 } } });
        emit("content_block_start", { index: 0, content_block: { type: "text", text: "" } });
        emit("content_block_delta", { index: 0, delta: { type: "text_delta", text: "OAUTH_OK" } });
        emit("content_block_stop", { index: 0 });
        emit("message_delta", { delta: { stop_reason: "end_turn", stop_sequence: null }, usage: { output_tokens: 10 } });
        emit("message_stop", {});
        res.end();
      } catch (error) {
        failures.push(error.message.replaceAll(token, "[synthetic token]"));
        res.writeHead(500);
        res.end('{"error":{"type":"api_error","message":"synthetic assertion failed"}}');
      }
    });
  });
  const proxy = createProviderServer(adapter);
  const originalRequest = http.request;
  try {
    await new Promise((resolve, reject) => {
      backend.once("error", reject);
      backend.listen(0, "127.0.0.1", resolve);
    });
    // AWF normalizes HTTP targets to port 80. Redirect only this mock
    // transport to its ephemeral port so the test needs no root privileges.
    http.request = function (options, ...args) {
      if (options.hostname === "127.0.0.1" && options.port === 80) {
        options = { ...options, port: backend.address().port };
      }
      return originalRequest.call(this, options, ...args);
    };
    await new Promise(resolve => proxy.listen(0, "127.0.0.1", resolve));
    // Only the loopback address changes for this experiment. AWF produced the
    // credential-free environment and its real HTTP proxy handles injection.
    agentEnv.ANTHROPIC_BASE_URL = `http://127.0.0.1:${proxy.address().port}`;
    agentEnv.CLAUDE_CONFIG_DIR = process.env.HOME;
    agentEnv.DISABLE_TELEMETRY = "1";
    agentEnv.DISABLE_ERROR_REPORTING = "1";
    agentEnv.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = "1";
    agentEnv.CLAUDE_CODE_DISABLE_FAST_MODE = "1";
    agentEnv.CLAUDE_CODE_MAX_RETRIES = "0";
    agentEnv.PATH = process.env.PATH;
    for (const key of ["HTTP_PROXY", "HTTPS_PROXY", "https_proxy", "CLAUDE_CODE_API_KEY_HELPER"]) delete agentEnv[key];
    const child = spawn(cli, ["--bare", "--no-session-persistence", "--tools", "", "--max-turns", "1", "--model", "claude-sonnet-4-6", "--output-format", "json", "--print", "Reply with OAUTH_OK."], { env: agentEnv, cwd: process.env.HOME });
    child.stdin.end();
    let stdout = "",
      stderr = "";
    child.stdout.on("data", chunk => {
      stdout += chunk;
    });
    child.stderr.on("data", chunk => {
      stderr += chunk;
    });
    const timer = setTimeout(() => child.kill("SIGKILL"), 45000);
    const code = await new Promise((resolve, reject) => {
      child.once("error", reject);
      child.once("exit", resolve);
    });
    clearTimeout(timer);
    assert.equal(failures.length, 0, failures.join("; "));
    assert.equal(code, 0, `synthetic CLI failed: ${stderr.replaceAll(token, "[synthetic token]")} ${stdout.replaceAll(token, "[synthetic token]")}`);
    assert(requests > 0, "the real CLI reached the AWF proxy");
    assert.equal(JSON.parse(stdout).result, "OAUTH_OK");
    process.stdout.write("PASS: compiled OAuth -> AWF isolation -> Claude CLI -> bearer proxy\n");
  } finally {
    http.request = originalRequest;
    proxy.shutdownConnections?.();
    proxy.closeAllConnections();
    backend.closeAllConnections();
    await Promise.all([new Promise(resolve => proxy.close(resolve)), new Promise(resolve => backend.close(resolve))]);
  }
}

main().catch(error => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
