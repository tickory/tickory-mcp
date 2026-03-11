#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { spawn } = require("node:child_process");
const { resolveRuntimeBinaryPath } = require("../scripts/lib/install");

const packageRoot = path.resolve(__dirname, "..");
const binaryPath = resolveRuntimeBinaryPath(packageRoot);

if (!fs.existsSync(binaryPath)) {
	process.stderr.write(
		`@tickory/mcp: missing binary at ${binaryPath}. Reinstall the package to download the platform build.\n`,
	);
	process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
	env: process.env,
	stdio: "inherit",
});

child.on("error", (error) => {
	process.stderr.write(`@tickory/mcp: failed to start ${binaryPath}: ${error.message}\n`);
	process.exit(1);
});

child.on("exit", (code, signal) => {
	if (signal) {
		process.stderr.write(`@tickory/mcp: binary exited with signal ${signal}\n`);
		process.exit(1);
	}

	process.exit(code ?? 0);
});
