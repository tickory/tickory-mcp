"use strict";

const supportedTargets = {
	darwin: {
		arm64: "arm64",
		x64: "amd64",
	},
	linux: {
		arm64: "arm64",
		x64: "amd64",
	},
};

function resolveTarget(platform = process.platform, arch = process.arch) {
	const arches = supportedTargets[platform];
	if (!arches) {
		throw new Error(`unsupported platform ${platform}; supported platforms are darwin and linux`);
	}

	const goarch = arches[arch];
	if (!goarch) {
		throw new Error(`unsupported architecture ${arch}; supported architectures are x64 and arm64`);
	}

	return {
		goarch,
		goos: platform,
		assetName: `tickory-mcp-${platform}-${goarch}`,
	};
}

module.exports = {
	resolveTarget,
};
