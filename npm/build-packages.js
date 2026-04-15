#!/usr/bin/env node

/**
 * Generates per-platform npm packages from goreleaser binary output.
 *
 * Usage: node build-packages.js <version>
 *
 * Expects goreleaser archives at:
 *   dist/devdash_<version>_<os>_<arch>/devdash
 *
 * Produces npm packages at:
 *   npm/devdash-<os>-<npm-arch>/
 *     package.json
 *     bin/devdash (binary)
 */

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const version = process.argv[2];
if (!version) {
  console.error("Usage: node build-packages.js <version>");
  process.exit(1);
}

const PLATFORMS = [
  {
    os: "darwin",
    arch: "arm64",
    npmArch: "arm64",
    goreleaseArch: "arm64",
    npmName: "devdash-darwin-arm64",
  },
  {
    os: "darwin",
    arch: "amd64",
    npmArch: "x64",
    goreleaseArch: "amd64",
    npmName: "devdash-darwin-x64",
  },
  {
    os: "linux",
    arch: "amd64",
    npmArch: "x64",
    goreleaseArch: "amd64",
    npmName: "devdash-linux-x64",
  },
  {
    os: "linux",
    arch: "arm64",
    npmArch: "arm64",
    goreleaseArch: "arm64",
    npmName: "devdash-linux-arm64",
  },
  {
    os: "windows",
    arch: "amd64",
    npmArch: "x64",
    goreleaseArch: "amd64",
    npmName: "devdash-win32-x64",
    binary: "devdash.exe",
  },
  {
    os: "windows",
    arch: "arm64",
    npmArch: "arm64",
    goreleaseArch: "arm64",
    npmName: "devdash-win32-arm64",
    binary: "devdash.exe",
  },
];

const distDir = path.join(__dirname, "..", "dist");

for (const platform of PLATFORMS) {
  const pkgDir = path.join(__dirname, platform.npmName);
  const binDir = path.join(pkgDir, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  // Find the binary from goreleaser output
  const binaryName = platform.binary || "devdash";
  const archiveDir = `devdash_${platform.os}_${platform.goreleaseArch}`;
  const binarySrc = path.join(distDir, archiveDir, binaryName);

  if (!fs.existsSync(binarySrc)) {
    // Try alternative naming (goreleaser v2 format)
    const altDir = `devdash_${platform.os}_${platform.goreleaseArch}_v1`;
    const altSrc = path.join(distDir, altDir, binaryName);
    if (fs.existsSync(altSrc)) {
      fs.copyFileSync(altSrc, path.join(binDir, binaryName));
    } else {
      console.warn(
        `Warning: binary not found for ${platform.os}/${platform.goreleaseArch}`
      );
      continue;
    }
  } else {
    fs.copyFileSync(binarySrc, path.join(binDir, binaryName));
  }

  fs.chmodSync(path.join(binDir, binaryName), 0o755);

  // Write package.json
  const npmOs = platform.os === "windows" ? "win32" : platform.os;
  const pkg = {
    name: `@devdashproject/${platform.npmName}`,
    version: version,
    description: `DevDash CLI binary for ${npmOs}/${platform.npmArch}`,
    os: [npmOs],
    cpu: [platform.npmArch],
    bin: {
      devdash: `bin/${binaryName}`,
    },
    repository: {
      type: "git",
      url: "https://github.com/devdashproject/devdash-cli",
    },
    license: "MIT",
  };

  fs.writeFileSync(
    path.join(pkgDir, "package.json"),
    JSON.stringify(pkg, null, 2) + "\n"
  );

  console.log(`Built: ${platform.npmName} (${platform.os}/${platform.npmArch})`);
}

// Update wrapper package version
const wrapperPkgPath = path.join(__dirname, "devdash-cli", "package.json");
const wrapperPkg = JSON.parse(fs.readFileSync(wrapperPkgPath, "utf8"));
wrapperPkg.version = version;
for (const dep of Object.keys(wrapperPkg.optionalDependencies || {})) {
  wrapperPkg.optionalDependencies[dep] = version;
}
fs.writeFileSync(wrapperPkgPath, JSON.stringify(wrapperPkg, null, 2) + "\n");
console.log(`Updated wrapper package to ${version}`);
