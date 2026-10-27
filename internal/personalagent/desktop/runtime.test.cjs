const test = require("node:test");
const assert = require("node:assert/strict");

const { defaultWorkspace, isSafeExternalURL, parseAgentURL } = require("./runtime.cjs");

test("default workspace stays inside Cockpit home", () => {
  assert.equal(defaultWorkspace("/home/alex"), "/home/alex/.cockpit/workspace");
});

test("startup output yields the loopback agent URL", () => {
  assert.equal(
    parseAgentURL("Cockpit Personal Agent\nWorkspace: /tmp/work\nOpen: http://127.0.0.1:41827\n"),
    "http://127.0.0.1:41827/",
  );
});

test("startup output without a ready address remains pending", () => {
  assert.equal(parseAgentURL("Cockpit Personal Agent\nWorkspace: /tmp/work\n"), null);
});

test("startup output cannot redirect Electron to a remote host", () => {
  assert.throws(
    () => parseAgentURL("Open: https://example.com"),
    /outside the loopback interface/,
  );
});

test("startup output cannot select a non-root local URL", () => {
  assert.throws(
    () => parseAgentURL("Open: http://127.0.0.1:41827/other"),
    /outside the loopback interface/,
  );
});

test("external Markdown links allow only web and mail protocols", () => {
  assert.equal(isSafeExternalURL("https://example.com"), true);
  assert.equal(isSafeExternalURL("mailto:team@example.com"), true);
  assert.equal(isSafeExternalURL("javascript:alert(1)"), false);
  assert.equal(isSafeExternalURL("file:///etc/passwd"), false);
});
