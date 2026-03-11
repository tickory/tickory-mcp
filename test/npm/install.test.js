"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const os = require("node:os");
const path = require("node:path");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const { once } = require("node:events");
const { spawn } = require("node:child_process");
const { installBinary, resolveReleaseInfo } = require("../../scripts/lib/install");

async function createTempPackageRoot(t, version = "1.2.3") {
	const packageRoot = await fsp.mkdtemp(path.join(os.tmpdir(), "tickory-mcp-package-"));
	t.after(async () => {
		await fsp.rm(packageRoot, { recursive: true, force: true });
	});

	await fsp.writeFile(
		path.join(packageRoot, "package.json"),
		JSON.stringify(
			{
				name: "@tickory/mcp",
				version,
			},
			null,
			2,
		),
	);

	return packageRoot;
}

test("resolveReleaseInfo derives the GitHub release asset from the package version", async (t) => {
	const packageRoot = await createTempPackageRoot(t, "2.3.4");

	assert.deepEqual(
		resolveReleaseInfo({
			packageRoot,
			platform: "darwin",
			arch: "arm64",
		}),
		{
			assetName: "tickory-mcp-darwin-arm64",
			releaseTag: "v2.3.4",
			assetURL: "https://github.com/tickory/tickory-mcp/releases/download/v2.3.4/tickory-mcp-darwin-arm64",
		},
	);
});

test("installBinary downloads the platform asset and makes it executable", async (t) => {
	const packageRoot = await createTempPackageRoot(t, "1.2.3");
	const requests = [];
	const baseURL = "https://example.test/releases/v1.2.3";

	const result = await installBinary({
		packageRoot,
		env: {
			TICKORY_MCP_RELEASE_BASE_URL: baseURL,
		},
		fetchImpl: async (url) => {
			requests.push(url);
			return new Response("#!/bin/sh\necho tickory-mcp\n", {
				status: 200,
				headers: {
					"content-type": "application/octet-stream",
				},
			});
		},
		platform: "linux",
		arch: "x64",
	});

	assert.equal(requests.length, 1);
	assert.equal(requests[0], `${baseURL}/tickory-mcp-linux-amd64`);
	assert.equal(result.assetURL, `${baseURL}/tickory-mcp-linux-amd64`);

	const binary = await fsp.readFile(result.binaryPath, "utf8");
	assert.match(binary, /tickory-mcp/);

	const mode = fs.statSync(result.binaryPath).mode & 0o777;
	assert.equal(mode, 0o755);
});

test("cli wrapper executes the installed binary and forwards arguments", async (t) => {
	const packageRoot = await createTempPackageRoot(t);
	const binaryPath = path.join(packageRoot, "tickory-mcp-test-binary");
	const argsFile = path.join(packageRoot, "args.txt");
	const envFile = path.join(packageRoot, "env.txt");

	await fsp.writeFile(
		binaryPath,
		[
			"#!/bin/sh",
			"printf '%s\\n' \"$@\" > \"$TICKORY_MCP_TEST_ARGS_FILE\"",
			"printf '%s' \"$TICKORY_API_KEY\" > \"$TICKORY_MCP_TEST_ENV_FILE\"",
		].join("\n"),
	);
	await fsp.chmod(binaryPath, 0o755);

	const cliPath = path.resolve(__dirname, "../../bin/tickory-mcp.js");
	const child = spawn(process.execPath, [cliPath, "--timeout", "3s"], {
		env: {
			...process.env,
			TICKORY_API_KEY: "tk_test_key",
			TICKORY_MCP_BINARY_PATH: binaryPath,
			TICKORY_MCP_TEST_ARGS_FILE: argsFile,
			TICKORY_MCP_TEST_ENV_FILE: envFile,
		},
	});

	const [code, signal] = await once(child, "exit");
	assert.equal(signal, null);
	assert.equal(code, 0);
	assert.equal(await fsp.readFile(argsFile, "utf8"), "--timeout\n3s\n");
	assert.equal(await fsp.readFile(envFile, "utf8"), "tk_test_key");
});
