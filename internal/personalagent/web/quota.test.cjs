const test = require("node:test");
const assert = require("node:assert/strict");
const { providerAvailable, providerQuotaLabel } = require("./quota.js");

test("only confirmed exhaustion blocks supported providers", () => {
  for (const status of ["available", "unknown", undefined]) {
    assert.equal(providerAvailable({ supported: true, quota: { status } }), true);
  }
  assert.equal(providerAvailable({ supported: true, quota: { status: "exhausted" } }), false);
  assert.equal(providerAvailable({ supported: false }), false);
  assert.equal(providerAvailable(null), false);
});

test("quota labels distinguish unknown, available, credits and reset", () => {
  assert.equal(providerQuotaLabel({}), "Cota desconhecida");
  assert.equal(providerQuotaLabel({ quota: { status: "exhausted" } }), "Cota esgotada");
  const label = providerQuotaLabel({ quota: {
    status: "available", balance: "12.5", checked_at: "2026-10-27T12:00:00Z",
    windows: [{ name: "primary", remaining_percent: 75, resets_at: "2026-10-27T15:00:00Z" }],
  } });
  for (const text of ["Cota disponível", "75% restante", "renova", "Créditos: 12.5", "Consultado"]) assert.ok(label.includes(text));
  assert.ok(providerQuotaLabel({ quota: { unlimited: true } }).includes("Créditos ilimitados"));
  assert.ok(providerQuotaLabel({ quota: { windows: [{ name: "secondary", remaining_percent: 0 }] } }).includes("0% restante"));
});
