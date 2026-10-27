const test = require("node:test");
const assert = require("node:assert/strict");

const { render } = require("./markdown.js");

test("renders common Markdown blocks and inline formatting", () => {
  const html = render("# Título\n\n**forte** e `código`\n\n- um\n- dois\n\n```go\nfmt.Println(\"oi\")\n```");
  assert.match(html, /<h1>Título<\/h1>/);
  assert.match(html, /<strong>forte<\/strong>/);
  assert.match(html, /<code>código<\/code>/);
  assert.match(html, /<ul><li>um<\/li><li>dois<\/li><\/ul>/);
  assert.match(html, /<pre><code class="language-go">fmt\.Println\(&quot;oi&quot;\)<\/code><\/pre>/);
});

test("escapes raw HTML and refuses unsafe link protocols", () => {
  const html = render('<script>alert("x")</script> [perigo](javascript:alert(1))');
  assert.equal(html, "<p>&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt; [perigo](javascript:alert(1))</p>");
  assert.doesNotMatch(html, /<script>|href="javascript:/);
});

test("renders only safe links with external-window protections", () => {
  assert.equal(
    render("[site](https://example.com)"),
    '<p><a href="https://example.com" target="_blank" rel="noopener noreferrer">site</a></p>',
  );
});

test("renders Markdown tables and alignment without interpreting HTML", () => {
  const html = render("| Name | Value |\n|:---:|---:|\n| `<img>` | **42** |");
  assert.match(html, /<th class="align-center">Name<\/th>/);
  assert.match(html, /<th class="align-right">Value<\/th>/);
  assert.match(html, /&lt;img&gt;/);
  assert.match(html, /<strong>42<\/strong>/);
  assert.doesNotMatch(html, /<img>/);
});
