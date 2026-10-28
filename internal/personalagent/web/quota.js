(function (root) {
  function providerAvailable(provider) {
    return Boolean(provider?.supported && provider.quota?.status !== "exhausted");
  }

  function providerQuotaLabel(provider) {
    const quota = provider.quota;
    const parts = [quota?.status === "exhausted" ? "Cota esgotada" : quota?.status === "available" ? "Cota disponível" : "Cota desconhecida"];
    for (const window of quota?.windows || []) {
      let label = `${window.name}: ${window.remaining_percent}% restante`;
      if (window.resets_at) label += ` · renova ${new Date(window.resets_at).toLocaleString("pt-BR")}`;
      parts.push(label);
    }
    if (quota?.unlimited) parts.push("Créditos ilimitados");
    else if (quota?.balance != null) parts.push(`Créditos: ${quota.balance}`);
    if (quota?.checked_at) parts.push(`Consultado ${new Date(quota.checked_at).toLocaleString("pt-BR")}`);
    return parts.join(" · ");
  }

  const api = { providerAvailable, providerQuotaLabel };
  if (typeof module !== "undefined" && module.exports) module.exports = api;
  else root.CockpitQuota = api;
})(globalThis);
