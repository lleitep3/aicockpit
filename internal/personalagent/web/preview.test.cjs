const test = require("node:test");
const assert = require("node:assert/strict");
const { previewURL, createPreviewPanel } = require("./preview.js");

for (const value of ["javascript:alert(1)", "data:text/html,test", "file:///tmp/test", "mailto:test@example.com", "https://user:pass@example.com", "not a URL"]) {
  test(`preview rejects unsafe or unsupported URL: ${value}`, () => {
    assert.equal(previewURL(value), null);
  });
}

for (const value of ["https://example.com/page?q=1#section", "http://localhost:3000/"]) {
  test(`preview accepts web URL: ${value}`, () => {
    assert.equal(previewURL(value), value);
  });
}

function fixture() {
  const elements = new Map();
  for (const id of ["link-preview", "link-preview-toggle", "link-preview-close", "link-preview-frame", "link-preview-title", "link-preview-external"]) {
    const classes = new Set(["hidden"]);
    elements.set(id, {
      classList: { toggle(name, on) { on ? classes.add(name) : classes.delete(name); }, contains(name) { return classes.has(name); } },
      attributes: {},
      setAttribute(name, value) { this.attributes[name] = value; },
      addEventListener(name, handler) { this[name] = handler; },
      focus() { this.focused = true; },
    });
  }
  const shell = { classList: { toggle() {} } };
  const doc = { getElementById: (id) => elements.get(id), querySelector: () => shell };
  return { elements, panel: createPreviewPanel(doc) };
}

test("opening a link exposes panel and safe external fallback", () => {
  const { panel, elements } = fixture();
  assert.equal(panel.open("https://example.com/", "Reference"), true);
  assert.equal(elements.get("link-preview-frame").src, "https://example.com/");
  assert.equal(elements.get("link-preview-title").textContent, "Reference");
  assert.equal(elements.get("link-preview-external").href, "https://example.com/");
  assert.equal(elements.get("link-preview-toggle").attributes["aria-expanded"], "true");
  assert.equal(elements.get("link-preview-close").focused, true);
});

test("toggle and close preserve frame URL and restore focus", () => {
  const { panel, elements } = fixture();
  panel.open("http://localhost:3000/");
  elements.get("link-preview-close").click();
  assert.equal(elements.get("link-preview").classList.contains("hidden"), true);
  assert.equal(elements.get("link-preview-toggle").focused, true);
  assert.equal(elements.get("link-preview-toggle").attributes["aria-expanded"], "false");
  elements.get("link-preview-toggle").click();
  assert.equal(elements.get("link-preview").classList.contains("hidden"), false);
  assert.equal(elements.get("link-preview-frame").src, "http://localhost:3000/");
});

test("Escape closes panel and restores toggle focus; other keys leave it open", () => {
  const { panel, elements } = fixture();
  panel.open("https://example.com/");
  const element = elements.get("link-preview");
  element.keydown({ key: "Tab" });
  assert.equal(element.classList.contains("hidden"), false);
  let prevented = false;
  element.keydown({ key: "Escape", preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.equal(element.classList.contains("hidden"), true);
  assert.equal(elements.get("link-preview-toggle").focused, true);
});

test("invalid links do not replace selected page; empty toggle is safe", () => {
  const { panel, elements } = fixture();
  elements.get("link-preview-toggle").click();
  panel.open("https://example.com/");
  assert.equal(panel.open("javascript:alert(1)"), false);
  assert.equal(elements.get("link-preview-frame").src, "https://example.com/");
});

test("delegation intercepts regular web links but preserves modified clicks and mail", () => {
  const { panel } = fixture();
  function click(href, extra = {}) {
    const event = { button: 0, target: { closest: () => ({ href, textContent: "Page", hasAttribute: () => false }) }, preventDefault() { this.prevented = true; }, ...extra };
    panel.handleClick(event);
    return event.prevented === true;
  }
  assert.equal(click("https://example.com/"), true);
  for (const extra of [{ ctrlKey: true }, { metaKey: true }, { shiftKey: true }, { altKey: true }, { button: 1 }, { defaultPrevented: true }]) {
    assert.equal(click("https://example.com/", extra), false);
  }
  assert.equal(click("mailto:test@example.com"), false);
  assert.equal(click("https://example.com/", { target: { closest: () => null } }), false);
});
