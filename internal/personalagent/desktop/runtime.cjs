const path = require("node:path");

function defaultWorkspace(homePath) {
  if (typeof homePath !== "string" || homePath.length === 0) {
    throw new TypeError("A home directory is required.");
  }
  return path.join(homePath, ".cockpit", "workspace");
}

function parseAgentURL(output) {
  const match = /^Open:\s*(\S+)\s*$/m.exec(output);
  if (!match) return null;

  let url;
  try {
    url = new URL(match[1]);
  } catch (error) {
    throw new Error("The local agent returned an invalid address.", { cause: error });
  }

  if (
    url.protocol !== "http:" ||
    url.hostname !== "127.0.0.1" ||
    !url.port ||
    url.pathname !== "/" ||
    url.search ||
    url.hash ||
    url.username ||
    url.password
  ) {
    throw new Error("The local agent returned an address outside the loopback interface.");
  }

  return url.href;
}

function isSafeExternalURL(value) {
  try {
    const url = new URL(value);
    if (url.protocol === "mailto:") return Boolean(url.pathname);
    return (url.protocol === "http:" || url.protocol === "https:") && Boolean(url.hostname);
  } catch {
    return false;
  }
}

module.exports = { defaultWorkspace, isSafeExternalURL, parseAgentURL };
