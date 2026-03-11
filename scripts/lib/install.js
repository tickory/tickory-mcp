"use strict";

const fs = require("node:fs");
const fsp = require("node:fs/promises");
const path = require("node:path");
const { Readable } = require("node:stream");
const { pipeline } = require("node:stream/promises");
const { resolveTarget } = require("./platform");

function readPackageVersion(packageRoot) {
	const packageJSONPath = path.join(packageRoot, "package.json");
	const packageJSON = JSON.parse(fs.readFileSync(packageJSONPath, "utf8"));
	return packageJSON.version;
}

function resolveBinaryPath(packageRoot) {
	return path.join(packageRoot, "vendor", "tickory-mcp");
}

function resolveRuntimeBinaryPath(packageRoot, env = process.env) {
	if (env.TICKORY_MCP_BINARY_PATH) {
		return path.resolve(env.TICKORY_MCP_BINARY_PATH);
	}

	return resolveBinaryPath(packageRoot);
}

function resolveReleaseInfo({ packageRoot, env = process.env, platform = process.platform, arch = process.arch }) {
	const { assetName } = resolveTarget(platform, arch);
	const version = readPackageVersion(packageRoot);
	const releaseTag = env.TICKORY_MCP_RELEASE_TAG || `v${version}`;
	const baseURL =
		env.TICKORY_MCP_RELEASE_BASE_URL ||
		`https://github.com/tickory/tickory-mcp/releases/download/${releaseTag}`;

	return {
		assetName,
		releaseTag,
		assetURL: `${baseURL.replace(/\/+$/, "")}/${assetName}`,
	};
}

function responseBodyToNodeStream(body) {
	if (!body) {
		throw new Error("download response did not include a body");
	}

	if (typeof body.pipe === "function") {
		return body;
	}

	if (typeof Readable.fromWeb === "function") {
		return Readable.fromWeb(body);
	}

	throw new Error("download response body is not a supported stream");
}

async function installBinary({
	packageRoot,
	env = process.env,
	fetchImpl = globalThis.fetch,
	platform = process.platform,
	arch = process.arch,
}) {
	if (typeof fetchImpl !== "function") {
		throw new Error("global fetch is unavailable; use Node.js 18 or newer");
	}

	const binaryPath = resolveBinaryPath(packageRoot);
	const { assetName, assetURL } = resolveReleaseInfo({
		packageRoot,
		env,
		platform,
		arch,
	});

	await fsp.mkdir(path.dirname(binaryPath), { recursive: true });

	const tempPath = `${binaryPath}.tmp`;
	try {
		const response = await fetchImpl(assetURL, {
			headers: {
				"user-agent": "@tickory/mcp",
			},
			redirect: "follow",
		});

		if (!response.ok) {
			throw new Error(`download failed for ${assetName}: ${response.status} ${response.statusText}`);
		}

		const bodyStream = responseBodyToNodeStream(response.body);
		await pipeline(bodyStream, fs.createWriteStream(tempPath, { mode: 0o755 }));
		await fsp.chmod(tempPath, 0o755);
		await fsp.rename(tempPath, binaryPath);
	} catch (error) {
		await fsp.rm(tempPath, { force: true });
		throw error;
	}

	return {
		assetName,
		assetURL,
		binaryPath,
	};
}

module.exports = {
	installBinary,
	readPackageVersion,
	resolveBinaryPath,
	resolveRuntimeBinaryPath,
	resolveReleaseInfo,
};
