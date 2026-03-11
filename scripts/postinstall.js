#!/usr/bin/env node
"use strict";

const path = require("node:path");
const { installBinary } = require("./lib/install");

async function main() {
	if (process.env.TICKORY_MCP_SKIP_POSTINSTALL === "1") {
		process.stdout.write("@tickory/mcp: skipping postinstall download\n");
		return;
	}

	const packageRoot = path.resolve(__dirname, "..");
	const result = await installBinary({ packageRoot });
	process.stdout.write(`@tickory/mcp: installed ${result.assetName}\n`);
}

main().catch((error) => {
	process.stderr.write(`@tickory/mcp: ${error.message}\n`);
	process.exit(1);
});
