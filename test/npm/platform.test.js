"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { resolveTarget } = require("../../scripts/lib/platform");

test("resolveTarget maps supported platforms to release asset names", () => {
	assert.deepEqual(resolveTarget("darwin", "arm64"), {
		goarch: "arm64",
		goos: "darwin",
		assetName: "tickory-mcp-darwin-arm64",
	});

	assert.deepEqual(resolveTarget("linux", "x64"), {
		goarch: "amd64",
		goos: "linux",
		assetName: "tickory-mcp-linux-amd64",
	});
});

test("resolveTarget rejects unsupported platforms and architectures", () => {
	assert.throws(() => resolveTarget("win32", "x64"), /unsupported platform win32/);
	assert.throws(() => resolveTarget("linux", "ia32"), /unsupported architecture ia32/);
});
