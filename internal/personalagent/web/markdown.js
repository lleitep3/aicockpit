(function exposeMarkdown(global) {
  function escapeHTML(value) {
    return value.replace(/[&<>"']/g, (character) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    })[character]);
  }

  function inlineMarkdown(value, allowLinks = true) {
    const token = /\[([^\]]+)\]\((https?:\/\/[^)\s]+|mailto:[^)\s]+)\)|\*\*([^*]+)\*\*|__([^_]+)__|~~([^~]+)~~|`([^`]+)`|\*([^*\n]+)\*|_([^_\n]+)_|(https?:\/\/[^\s<>`"']+)/g;
    let output = "";
    let cursor = 0;
    for (const match of value.matchAll(token)) {
      output += escapeHTML(value.slice(cursor, match.index));
      if (!allowLinks && (match[1] !== undefined || match[9] !== undefined)) {
        output += escapeHTML(match[0]);
      } else if (match[1] !== undefined) {
        output += `<a href="${escapeHTML(match[2])}" target="_blank" rel="noopener noreferrer">${inlineMarkdown(match[1], false)}</a>`;
      } else if (match[3] !== undefined || match[4] !== undefined) {
        output += `<strong>${inlineMarkdown(match[3] ?? match[4], allowLinks)}</strong>`;
      } else if (match[5] !== undefined) {
        output += `<del>${inlineMarkdown(match[5], allowLinks)}</del>`;
      } else if (match[6] !== undefined) {
        output += `<code>${escapeHTML(match[6])}</code>`;
      } else if (match[9] !== undefined) {
        const url = match[9].replace(/[.,;:!?)\]]+$/, "");
        output += `<a href="${escapeHTML(url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(url)}</a>${escapeHTML(match[9].slice(url.length))}`;
      } else {
        output += `<em>${escapeHTML(match[7] ?? match[8])}</em>`;
      }
      cursor = match.index + match[0].length;
    }
    return output + escapeHTML(value.slice(cursor));
  }

  function render(markdown) {
    const lines = String(markdown).replace(/\r\n?/g, "\n").split("\n");
    const output = [];
    let index = 0;
    while (index < lines.length) {
      const line = lines[index];
      if (!line.trim()) {
        index += 1;
        continue;
      }

      const fence = /^```([a-z0-9_+-]*)\s*$/i.exec(line);
      if (fence) {
        const code = [];
        index += 1;
        while (index < lines.length && !/^```\s*$/.test(lines[index])) {
          code.push(lines[index]);
          index += 1;
        }
        if (index < lines.length) index += 1;
        const language = /^[a-z0-9_+-]+$/i.test(fence[1]) ? ` class="language-${fence[1]}"` : "";
        output.push(`<pre><code${language}>${escapeHTML(code.join("\n"))}</code></pre>`);
        continue;
      }

      const heading = /^(#{1,6})\s+(.+?)\s*#*\s*$/.exec(line);
      if (heading) {
        const level = heading[1].length;
        output.push(`<h${level}>${inlineMarkdown(heading[2])}</h${level}>`);
        index += 1;
        continue;
      }

      if (/^\s*(?:[-*_]\s*){3,}$/.test(line)) {
        output.push("<hr>");
        index += 1;
        continue;
      }

      const listItem = /^\s*([-*+])\s+(.+)$/.exec(line);
      const orderedItem = /^\s*\d+[.)]\s+(.+)$/.exec(line);
      if (listItem || orderedItem) {
        const ordered = Boolean(orderedItem);
        const items = [];
        while (index < lines.length) {
          const current = ordered
            ? /^\s*\d+[.)]\s+(.+)$/.exec(lines[index])
            : /^\s*([-*+])\s+(.+)$/.exec(lines[index]);
          if (!current) break;
          items.push(`<li>${inlineMarkdown(current[ordered ? 1 : 2])}</li>`);
          index += 1;
        }
        const tag = ordered ? "ol" : "ul";
        output.push(`<${tag}>${items.join("")}</${tag}>`);
        continue;
      }

      const separator = lines[index + 1];
      if (line.includes("|") && separator && isTableSeparator(separator)) {
        const headers = splitTableRow(line);
        const alignments = splitTableRow(separator).map((cell) => {
          if (cell.startsWith(":") && cell.endsWith(":")) return "center";
          return cell.endsWith(":") ? "right" : "left";
        });
        const renderCells = (cells, tag) => {
          const rendered = headers.map((_, cellIndex) => {
            const alignment = ` class="align-${alignments[cellIndex] || "left"}"`;
            return `<${tag}${alignment}>${inlineMarkdown(cells[cellIndex] || "")}</${tag}>`;
          });
          return `<tr>${rendered.join("")}</tr>`;
        };
        const rows = [];
        index += 2;
        while (index < lines.length && lines[index].trim() && lines[index].includes("|")) {
          rows.push(renderCells(splitTableRow(lines[index]), "td"));
          index += 1;
        }
        output.push(`<table><thead>${renderCells(headers, "th")}</thead><tbody>${rows.join("")}</tbody></table>`);
        continue;
      }

      if (/^\s*>\s?/.test(line)) {
        const quote = [];
        while (index < lines.length && /^\s*>\s?/.test(lines[index])) {
          quote.push(lines[index].replace(/^\s*>\s?/, ""));
          index += 1;
        }
        output.push(`<blockquote><p>${quote.map((line) => inlineMarkdown(line)).join("<br>")}</p></blockquote>`);
        continue;
      }

      function isTableSeparator(line) {
        const cells = splitTableRow(line);
        return cells.length > 1 && cells.every((cell) => /^:?-{3,}:?$/.test(cell));
      }

      function splitTableRow(row) {
        const cells = [];
        let cell = "";
        let escaped = false;
        for (const character of row.trim()) {
          if (character === "|" && !escaped) {
            cells.push(cell.trim());
            cell = "";
          } else {
            cell += character;
          }
          escaped = character === "\\" && !escaped;
          if (character !== "\\") escaped = false;
        }
        cells.push(cell.trim());
        if (cells[0] === "") cells.shift();
        if (cells[cells.length - 1] === "") cells.pop();
        return cells;
      }

      const paragraph = [];
      while (
        index < lines.length &&
        lines[index].trim() &&
        !/^```/.test(lines[index]) &&
        !/^(?:#{1,6})\s+/.test(lines[index]) &&
        !/^\s*(?:[-*_]\s*){3,}$/.test(lines[index]) &&
        !/^\s*(?:[-*+]|\d+[.)])\s+/.test(lines[index]) &&
        !/^\s*>\s?/.test(lines[index])
      ) {
        paragraph.push(lines[index]);
        index += 1;
      }
      output.push(`<p>${paragraph.map((line) => inlineMarkdown(line)).join("<br>")}</p>`);
    }
    return output.join("");
  }

  const api = { render };
  global.CockpitMarkdown = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(globalThis);
