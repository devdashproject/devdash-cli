const os = require("os");
const { execSync } = require("child_process");

const PLATFORM_PACKAGES = {
  "darwin-arm64": "@devdashproject/devdash-darwin-arm64",
  "darwin-x64": "@devdashproject/devdash-darwin-x64",
  "linux-x64": "@devdashproject/devdash-linux-x64",
  "linux-arm64": "@devdashproject/devdash-linux-arm64",
  "win32-x64": "@devdashproject/devdash-win32-x64",
  "win32-arm64": "@devdashproject/devdash-win32-arm64",
};

const key = `${os.platform()}-${os.arch()}`;
const pkg = PLATFORM_PACKAGES[key];

if (!pkg) {
  console.warn(
    `[devdash] Warning: no prebuilt binary for ${key}. You may need to build from source.`
  );
  process.exit(0);
}

try {
  require.resolve(`${pkg}/bin/devdash`);
} catch {
  // Optional dependency wasn't installed (e.g. npm bug or filtered platform)
  console.warn(
    `[devdash] Platform package ${pkg} not found. Attempting explicit install...`
  );
  try {
    execSync(`npm install ${pkg}`, { stdio: "inherit" });
  } catch {
    console.warn(
      `[devdash] Could not install ${pkg}. The CLI may not work on this platform.`
    );
  }
}
