(function (root) {
  const LOW_QUOTA_THRESHOLD_PERCENT = 30;

  function providerAvailable(provider) {
    return Boolean(provider?.supported && provider.quota?.status !== "exhausted");
  }

  function providerQuotaSummary(provider, now = new Date()) {
    const quota = provider?.quota || {};
    const windows = (quota.windows || []).filter((item) => {
      const remaining = item.remaining_percent;
      return Number.isFinite(remaining) && remaining >= 0 && remaining <= 100;
    });
    const low = !quota.unlimited && windows.some((item) => item.remaining_percent <= LOW_QUOTA_THRESHOLD_PERCENT);
    const state = quota.status === "exhausted"
      ? "exhausted"
      : quota.status === "available"
        ? low ? "low" : "available"
        : "unknown";
    const resetDates = windows
      .map((item) => new Date(item.resets_at))
      .filter((date) => Number.isFinite(date.getTime()) && date > now)
      .sort((first, second) => first - second);
    const nextReset = resetDates[0];
    const resetOptions = {
      day: "2-digit",
      month: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    };
    if (nextReset && nextReset.getFullYear() !== now.getFullYear()) resetOptions.year = "numeric";
    const balance = String(quota.balance || "").trim();
    const credits = quota.unlimited
      ? "Créditos ilimitados"
      : balance
        ? /cr[eé]dit/i.test(balance) ? balance : `Créditos ${balance}`
        : "";

    return {
      state,
      symbol: state === "low" ? "▲" : "●",
      label: {
        available: "Cota disponível",
        low: "Cota baixa",
        exhausted: "Cota esgotada",
        unknown: "Cota desconhecida",
      }[state],
      nextRefill: nextReset ? nextReset.toLocaleString("pt-BR", resetOptions) : "",
      credits,
    };
  }

  function modelQuotaLabel(provider, model) {
    if (provider?.id !== "antigravity") return "";
    const modelIdentity = `${model?.id || ""} ${model?.name || ""}`.toLocaleLowerCase("pt-BR");
    const quotaWindows = provider.quota?.windows || [];
    let windows = [];

    if (modelIdentity.includes("gemini")) {
      windows = quotaWindows.filter((item) => {
        const name = item.name.toLocaleLowerCase("pt-BR");
        return name.includes("gemini models") ||
          (modelIdentity.includes("gemini 4 argon") && name.includes("gemini 4 argon"));
      });
    } else if (/\bclaude\b|\bgpt\b/.test(modelIdentity)) {
      windows = quotaWindows.filter((item) => item.name.toLocaleLowerCase("pt-BR").includes("claude and gpt models"));
    }

    if (!windows.length) {
      return quotaWindows.length ? "Grupo de cota não identificado" : "Cota por modelo desconhecida";
    }
    return windows.map((item) => {
      const name = item.name.toLocaleLowerCase("pt-BR");
      const scope = name.includes("gemini 4 argon")
        ? "Argon"
        : name.includes("gemini models")
          ? "Gemini"
          : "Claude/GPT";
      const period = name.includes("five hour") ? "5h" : name.includes("weekly") ? "semanal" : "limite";
      let label = `${scope} ${period}: ${item.remaining_percent}% restante`;
      if (item.resets_at) {
        const reset = new Date(item.resets_at);
        if (!Number.isNaN(reset.getTime())) {
          label += ` · renova ${reset.toLocaleString("pt-BR", {
            day: "2-digit",
            month: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
          })}`;
        }
      }
      return label;
    }).join(" · ");
  }

  const api = { modelQuotaLabel, providerAvailable, providerQuotaSummary };
  if (typeof module !== "undefined" && module.exports) module.exports = api;
  else root.CockpitQuota = api;
})(globalThis);
