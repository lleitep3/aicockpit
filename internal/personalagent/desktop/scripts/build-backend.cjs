const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const desktopDirectory = path.resolve(__dirname, "..");
const repositoryRoot = path.resolve(desktopDirectory, "../../..");
const outputPath = path.join(desktopDirectory, "resources", "bin", "cockpit");
const architecture = { x64: "amd64", arm64: "arm64" }[process.arch];

if (process.platform !== "linux") {
  throw new Error("The Personal Agent installer currently targets Linux.");
}
if (!architecture) {
  throw new Error(`Unsupported Linux architecture: ${process.arch}`);
}

const version = fs.readFileSync(path.join(repositoryRoot, "VERSION"), "utf8").trim();
if (!/^\d+\.\d+\.\d+$/.test(version)) {
  throw new Error(`Repository VERSION is not a valid semantic version: ${version}`);
}

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
const result = spawnSync(
  "go",
  [
    "build",
    "-trimpath",
    `-ldflags=-s -w -X github.com/lleitep3/aicockpit/internal/version.Version=${version}`,
    "-o",
    outputPath,
    ".",
  ],
  {
    cwd: repositoryRoot,
    env: { ...process.env, GOOS: "linux", GOARCH: architecture },
    stdio: "inherit",
  },
);

if (result.error) throw result.error;
if (result.status !== 0) process.exit(result.status ?? 1);
fs.chmodSync(outputPath, 0o755);
process.stdout.write(`Built Linux ${architecture} backend: ${outputPath}\n`);
