const test = require("node:test");
const assert = require("node:assert/strict");
const { modelQuotaLabel, providerAvailable, providerQuotaSummary } = require("./quota.js");

test("only confirmed exhaustion blocks supported providers", () => {
  for (const status of ["available", "unknown", undefined]) {
    assert.equal(providerAvailable({ supported: true, quota: { status } }), true);
  }
  assert.equal(providerAvailable({ supported: true, quota: { status: "exhausted" } }), false);
  assert.equal(providerAvailable({ supported: false }), false);
  assert.equal(providerAvailable(null), false);
});

test("provider quota summary is compact and shows next refill and credits", () => {
  const now = new Date("2026-10-28T12:00:00Z");
  const summary = providerQuotaSummary({
    quota: {
      status: "available",
      balance: "12.5",
      windows: [
        { name: "weekly", remaining_percent: 31, resets_at: "2026-10-30T18:00:00Z" },
        { name: "daily", remaining_percent: 60, resets_at: "2026-10-29T13:00:00Z" },
        { name: "expired", remaining_percent: 80, resets_at: "2026-10-28T09:00:00Z" },
      ],
    },
  }, now);

  assert.deepEqual(
    { state: summary.state, symbol: summary.symbol, label: summary.label, credits: summary.credits },
    { state: "available", symbol: "●", label: "Cota disponível", credits: "Créditos 12.5" },
  );
  assert.match(summary.nextRefill, /29\/10/);
});

test("provider quota summary colors low, exhausted, unknown and unlimited states", () => {
  const now = new Date("2026-10-28T12:00:00Z");
  assert.deepEqual(
    (({ state, symbol, label }) => ({ state, symbol, label }))(
      providerQuotaSummary({ quota: { status: "available", windows: [{ remaining_percent: 30 }] } }, now),
    ),
    { state: "low", symbol: "▲", label: "Cota baixa" },
  );
  assert.deepEqual(
    (({ state, symbol, label }) => ({ state, symbol, label }))(
      providerQuotaSummary({ quota: { status: "exhausted", windows: [{ remaining_percent: 0 }] } }, now),
    ),
    { state: "exhausted", symbol: "●", label: "Cota esgotada" },
  );
  assert.deepEqual(
    (({ state, symbol, label }) => ({ state, symbol, label }))(
      providerQuotaSummary({ quota: { status: "unknown" } }, now),
    ),
    { state: "unknown", symbol: "●", label: "Cota desconhecida" },
  );
  const unlimited = providerQuotaSummary({
    quota: { status: "available", unlimited: true, windows: [{ remaining_percent: 0 }] },
  }, now);
  assert.equal(unlimited.state, "available");
  assert.equal(unlimited.credits, "Créditos ilimitados");
});

test("Antigravity model quota labels use only matching quota groups", () => {
  const provider = {
    id: "antigravity",
    quota: {
      status: "available",
      windows: [
        { name: "Gemini Models Weekly Limit Remaining", remaining_percent: 97, resets_at: "2026-11-03T17:30:06Z" },
        { name: "Gemini Models Five Hour Limit Remaining", remaining_percent: 100 },
        { name: "Gemini 4 Argon Gemini 4 Argon Weekly Limit", remaining_percent: 91, resets_at: "invalid" },
        { name: "Claude and GPT models Weekly Limit Remaining", remaining_percent: 88 },
        { name: "Claude and GPT models Five Hour Limit Remaining", remaining_percent: 99 },
      ],
    },
  };

  const argon = modelQuotaLabel(provider, { id: "gemini-4-argon-medium", name: "Gemini 4 Argon (Medium)" });
  assert.match(argon, /Gemini semanal: 97% restante/);
  assert.match(argon, /Gemini 5h: 100% restante/);
  assert.match(argon, /Argon semanal: 91% restante/);
  assert.match(argon, /renova/);
  assert.doesNotMatch(argon, /Invalid Date/);
  assert.doesNotMatch(argon, /Claude\/GPT/);

  const flash = modelQuotaLabel(provider, { id: "gemini-3.8-flash-high", name: "Gemini 3.8 Flash (High)" });
  assert.match(flash, /Gemini semanal: 97% restante/);
  assert.doesNotMatch(flash, /Argon/);

  const claude = modelQuotaLabel(provider, { id: "claude-sonnet-5-5-medium", name: "Claude Sonnet 5.5" });
  assert.match(claude, /Claude\/GPT semanal: 88% restante/);
  assert.match(claude, /Claude\/GPT 5h: 99% restante/);

  const gpt = modelQuotaLabel(provider, { id: "gpt-oss-120b-medium", name: "GPT-OSS 120B" });
  assert.match(gpt, /Claude\/GPT semanal: 88% restante/);
});

test("Antigravity model quota labels expose unknown or unmapped groups without guessing", () => {
  assert.equal(
    modelQuotaLabel({ id: "antigravity", quota: { status: "unknown" } }, { id: "gemini-4-argon-medium" }),
    "Cota por modelo desconhecida",
  );
  assert.equal(
    modelQuotaLabel({
      id: "antigravity",
      quota: { windows: [{ name: "SWE models weekly", remaining_percent: 80 }] },
    }, { id: "swe-2-high", name: "SWE-2 High" }),
    "Grupo de cota não identificado",
  );
  assert.equal(modelQuotaLabel({ id: "codex", quota: { windows: [] } }, { id: "gpt-5" }), "");
});
