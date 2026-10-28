(function exposePreview(global) {
  function previewURL(value) {
    try {
      const url = new URL(value);
      return ["http:", "https:"].includes(url.protocol) && !url.username && !url.password ? url.href : null;
    } catch {
      return null;
    }
  }

  function createPreviewPanel(doc) {
    const panel = doc.getElementById("link-preview");
    const toggle = doc.getElementById("link-preview-toggle");
    const close = doc.getElementById("link-preview-close");
    const frame = doc.getElementById("link-preview-frame");
    let selectedURL = "";
    let expanded = false;

    function setExpanded(value) {
      expanded = value && Boolean(selectedURL);
      panel.classList.toggle("hidden", !expanded);
      doc.querySelector(".app-shell").classList.toggle("preview-open", expanded);
      toggle.setAttribute("aria-expanded", String(expanded));
      if (expanded) close.focus();
      else toggle.focus();
    }

    function open(value, title) {
      const url = previewURL(value);
      if (!url) return false;
      if (selectedURL !== url) frame.src = url;
      selectedURL = url;
      const label = title?.trim() || url;
      doc.getElementById("link-preview-title").textContent = label;
      frame.title = `Visualização: ${label}`;
      doc.getElementById("link-preview-external").href = url;
      toggle.classList.toggle("hidden", false);
      setExpanded(true);
      return true;
    }

    function handleClick(event) {
      if (event.defaultPrevented || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
      const link = event.target.closest?.("a[href]");
      if (!link || link.hasAttribute("download") || !previewURL(link.href)) return;
      event.preventDefault();
      open(link.href, link.textContent);
    }

    toggle.addEventListener("click", () => setExpanded(!expanded));
    close.addEventListener("click", () => setExpanded(false));
    panel.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setExpanded(false);
      }
    });
    return { open, handleClick };
  }

  const api = { previewURL, createPreviewPanel };
  global.CockpitPreview = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(globalThis);
