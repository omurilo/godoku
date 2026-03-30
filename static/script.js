function updateExampleVisibility(lang) {
  if (!lang) {
    var sel = document.querySelector(".example-lang-select");
    if (sel) lang = sel.value;
  }

  var ids = ["curl", "go", "python", "js"];
  ids.forEach(function(id) {
    var el = document.getElementById("example-"+id);
    if (el) el.style.display = (id === lang) ? "block" : "none";
  });
}

window.showApiExample = function(sel) {
  var lang = sel.value;
  if (typeof processCodeBlocks === "function") {
    var visible = document.querySelector("#example-"+lang+" code");
    if (visible) processCodeBlocks([visible]);
  }
  setTimeout(function() { updateExampleVisibility(lang); }, 0);
}
// Godoku - client-side enhancements
document.addEventListener("DOMContentLoaded", function () {
    setTimeout(function() {
      updateExampleVisibility("curl");
    }, 0);
  // Theme toggle (desktop + mobile)
  function handleThemeToggle() {
    var current = document.documentElement.getAttribute("data-theme");
    var next = current === "light" ? "" : "light";
    if (next) {
      document.documentElement.setAttribute("data-theme", next);
    } else {
      document.documentElement.removeAttribute("data-theme");
    }
    localStorage.setItem("godoku-theme", next || "dark");

    // Update Shiki theme for already-highlighted blocks
    updateShikiTheme(next || "dark");
  }

  var themeToggle = document.getElementById("theme-toggle");
  if (themeToggle) themeToggle.addEventListener("click", handleThemeToggle);
  var mobileThemeToggle = document.getElementById("mobile-theme-toggle");
  if (mobileThemeToggle) mobileThemeToggle.addEventListener("click", handleThemeToggle);

  // Highlight active nav link
  const currentPath = window.location.pathname;
  document.querySelectorAll(".nav-link").forEach(function (link) {
    const href = link.getAttribute("href");
    if (currentPath.startsWith(href) && href !== "/") {
      link.classList.add("active");
    } else if (href === "/" && currentPath === "/") {
      link.classList.add("active");
    }
  });

  // Smooth scroll for anchor links
  document.querySelectorAll('a[href^="#"]').forEach(function (anchor) {
    anchor.addEventListener("click", function (e) {
      var id = this.getAttribute("href").substring(1);
      var target = document.getElementById(id);
      if (target) {
        e.preventDefault();
        target.scrollIntoView({ behavior: "smooth" });
        history.pushState(null, null, "#" + id);
      }
    });
  });

  // Collapsible sidebar groups
  document.querySelectorAll(".sidebar-group").forEach(function (group) {
    var toggle = group.querySelector(".sidebar-group-toggle");
    if (!toggle) return;
    var hasActive = group.querySelector(".sidebar-link.active");

    if (!hasActive) {
      group.classList.add("collapsed");
      toggle.setAttribute("aria-expanded", "false");
    }
  });

  document.querySelectorAll(".sidebar-group-toggle").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var group = btn.closest(".sidebar-group");
      var isCollapsed = group.classList.toggle("collapsed");
      btn.setAttribute("aria-expanded", isCollapsed ? "false" : "true");
    });
  });

  // Mobile menu toggle
  var sidebarToggle = document.getElementById("sidebar-toggle");
  var sidebar = document.querySelector(".sidebar");
  var mobileNav = document.getElementById("mobile-nav");

  if (sidebarToggle) {
    sidebarToggle.addEventListener("click", function () {
      var isOpen = sidebarToggle.classList.toggle("active");
      if (mobileNav) mobileNav.classList.toggle("open", isOpen);
      if (sidebar) sidebar.classList.toggle("sidebar-open", isOpen);
    });
  }

  // Banner dismiss
  var bannerClose = document.getElementById("banner-close");
  if (bannerClose) {
    var banner = document.getElementById("site-banner");
    if (sessionStorage.getItem("godoku-banner-dismissed")) {
      banner.classList.add("hidden");
    }
    bannerClose.addEventListener("click", function () {
      banner.classList.add("hidden");
      sessionStorage.setItem("godoku-banner-dismissed", "1");
    });
  }

  // TOC active tracking
  var tocLinks = document.querySelectorAll(".toc-link");
  if (tocLinks.length > 0) {
    var headingEls = [];
    tocLinks.forEach(function (link) {
      var href = link.getAttribute("href");
      if (!href) return;
      var id = href.substring(1);
      var el = document.getElementById(id);
      if (el) headingEls.push({ el: el, link: link });
    });

    function updateTOC() {
      var scrollY = window.scrollY + 100;
      var active = null;
      for (var i = headingEls.length - 1; i >= 0; i--) {
        if (headingEls[i].el.offsetTop <= scrollY) {
          active = headingEls[i].link;
          break;
        }
      }
      tocLinks.forEach(function (l) { l.classList.remove("active"); });
      if (active) active.classList.add("active");
    }

    window.addEventListener("scroll", updateTOC, { passive: true });
    updateTOC();
  }

  // Shiki code highlighting, then copy buttons
  initShikiHighlighting().then(function () {
    addCopyButtons();
  }).catch(function (err) {
    console.warn("Shiki failed, using plain code blocks:", err);
    addCopyButtons();
  }).finally(function () {
    updateExampleVisibility("curl");
  });
});

// Parse highlight range string "1,3-5,8" into a Set of line numbers
function parseHighlightRanges(str) {
  var result = new Set();
  if (!str) return result;
  str.split(",").forEach(function (part) {
    part = part.trim();
    var dashIdx = part.indexOf("-");
    if (dashIdx >= 0) {
      var start = parseInt(part.substring(0, dashIdx), 10);
      var end = parseInt(part.substring(dashIdx + 1), 10);
      for (var i = start; i <= end; i++) result.add(i);
    } else {
      var n = parseInt(part, 10);
      if (n > 0) result.add(n);
    }
  });
  return result;
}

// Shiki highlighter instance (shared)
var shikiHighlighter = null;

function getShikiTheme() {
  var theme = document.documentElement.getAttribute("data-theme");
  return theme === "light" ? "github-light" : "dracula";
}

async function initShikiHighlighting() {
  var codeBlocks = document.querySelectorAll("pre > code");
  if (codeBlocks.length === 0) return;

  // Collect languages from code blocks
  var langs = new Set();
  codeBlocks.forEach(function (code) {
    var cls = Array.from(code.classList).find(function (c) { return c.startsWith("language-"); });
    if (cls) langs.add(cls.replace("language-", ""));
    var pre = code.parentElement;
    if (pre && pre.dataset.lang) langs.add(pre.dataset.lang);
  });

  var langList = Array.from(langs).filter(function (l) { return l && l !== ""; });

  // Languages (and aliases) available in shiki@1/bundle/web
  var webBundleLangs = new Set([
    "angular-html", "angular-ts", "astro", "blade", "c", "coffee", "coffeescript",
    "cpp", "c++", "css", "glsl", "graphql", "gql", "haml", "handlebars", "hbs",
    "html", "html-derivative", "http", "imba", "java", "javascript", "js",
    "jinja", "jison", "json", "json5", "jsonc", "jsonl", "jsx", "julia", "jl",
    "less", "markdown", "md", "marko", "mdc", "mdx", "php", "postcss", "pug",
    "jade", "python", "py", "r", "regexp", "regex", "sass", "scss",
    "shellscript", "bash", "sh", "shell", "zsh", "sql", "stylus", "styl",
    "svelte", "ts-tags", "lit", "tsx", "typescript", "ts", "vue", "vue-html",
    "wasm", "wgsl", "xml", "yaml", "yml"
  ]);
  var useWebBundle = langList.length === 0 || langList.every(function (l) { return webBundleLangs.has(l); });
  var shiki = await import(useWebBundle ? "https://esm.sh/shiki@1/bundle/web" : "https://esm.sh/shiki@1/bundle/full");
  
  shikiHighlighter = await shiki.createHighlighter({
    themes: ["dracula", "github-light"],
    langs: langList.length > 0 ? langList : ["text"]
  });

  processCodeBlocks(codeBlocks);
}

function processCodeBlocks(codeBlocks) {
  if (!shikiHighlighter) return;

  var theme = getShikiTheme();
  var loadedLangs = shikiHighlighter.getLoadedLanguages();

  codeBlocks.forEach(function (codeEl) {
    var pre = codeEl.parentElement;
    if (!pre || pre.closest(".playground-result")) return;

    // Determine language
    var cls = Array.from(codeEl.classList).find(function (c) { return c.startsWith("language-"); });
    var lang = (cls ? cls.replace("language-", "") : "") || (pre.dataset.lang || "");
    if (!lang || loadedLangs.indexOf(lang) === -1) lang = "text";

    // Use stored original code on re-renders (theme toggle) to avoid
    // picking up line-number text that was injected into the DOM.
    var code;
    if (pre.dataset.originalCode) {
      code = pre.dataset.originalCode;
    } else {
      code = codeEl.textContent;
      if (code.endsWith("\n")) code = code.slice(0, -1);
    }

    var highlightStr = pre.dataset.highlight || "";
    var highlightLines = parseHighlightRanges(highlightStr);
    var showLineNumbers = pre.hasAttribute("data-line-numbers");

    // Save data attrs and id before replacing
    var savedAttrs = {};
    if (pre.dataset.highlight) savedAttrs.highlight = pre.dataset.highlight;
    if (pre.dataset.lineNumbers) savedAttrs.lineNumbers = pre.dataset.lineNumbers;
    if (pre.dataset.lang) savedAttrs.lang = pre.dataset.lang;
    if (pre.id) savedAttrs.id = pre.id;

    try {
      var html = shikiHighlighter.codeToHtml(code, {
        lang: lang,
        theme: theme
      });

      var temp = document.createElement("div");
      temp.innerHTML = html;
      var newPre = temp.querySelector("pre");
      if (!newPre) return;

      // Manually add line enhancements (line numbers + highlights)
      if (highlightLines.size > 0 || showLineNumbers) {
        var lineSpans = newPre.querySelectorAll("code > .line");
        lineSpans.forEach(function (span, idx) {
          var lineNum = idx + 1;
          if (highlightLines.has(lineNum)) {
            span.classList.add("highlighted");
          }
          if (showLineNumbers) {
            var numSpan = document.createElement("span");
            numSpan.className = "line-number";
            numSpan.textContent = String(lineNum);
            span.insertBefore(numSpan, span.firstChild);
          }
        });
      }

      // Restore data attributes and id
      if (savedAttrs.highlight) newPre.dataset.highlight = savedAttrs.highlight;
      if (savedAttrs.lineNumbers) newPre.dataset.lineNumbers = savedAttrs.lineNumbers;
      if (savedAttrs.lang) newPre.dataset.lang = savedAttrs.lang;
      if (savedAttrs.id) newPre.id = savedAttrs.id;
      newPre.dataset.shiki = "true";
      newPre.dataset.originalCode = code;

      // If inside a titled wrapper, just replace the pre
      var titled = pre.closest(".code-block-titled");
      if (titled) {
        pre.replaceWith(newPre);
      } else {
        pre.replaceWith(newPre);
      }
    } catch (e) {
      console.warn("Shiki error for lang '" + lang + "':", e);
    }
  });
}

function updateShikiTheme(themeMode) {
  if (!shikiHighlighter) return;
  var codeBlocks = document.querySelectorAll('pre[data-shiki="true"] > code');
  if (codeBlocks.length === 0) {
    codeBlocks = document.querySelectorAll("pre.shiki > code");
  }
  if (codeBlocks.length === 0) return;
  processCodeBlocks(codeBlocks);
  updateExampleVisibility();
}

function addCopyButtons() {
  document.querySelectorAll("pre").forEach(function (pre) {
    if (pre.closest(".playground-result")) return;
    if (pre.closest(".code-block-wrapper")) return;

    var wrapper = document.createElement("div");
    wrapper.className = "code-block-wrapper";
    pre.parentNode.insertBefore(wrapper, pre);
    wrapper.appendChild(pre);

    var toolbar = document.createElement("div");
    toolbar.className = "code-block-toolbar";

    var lang = pre.dataset.lang || "";
    if (lang) {
      var langLabel = document.createElement("span");
      langLabel.className = "code-lang-label";
      langLabel.textContent = lang;
      toolbar.appendChild(langLabel);
    }

    var copyBtn = document.createElement("button");
    copyBtn.className = "code-copy-btn";
    copyBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>';
    copyBtn.title = "Copy code";
    toolbar.appendChild(copyBtn);
    wrapper.appendChild(toolbar);

    copyBtn.addEventListener("click", function () {
      var code = pre.querySelector("code");
      var text = code ? code.textContent : pre.textContent;
      navigator.clipboard.writeText(text).then(function () {
        copyBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>';
        setTimeout(function () {
          copyBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>';
        }, 2000);
      });
    });
  });
}

// API Playground
function sendPlaygroundRequest(btn) {
  const playground = btn.closest(".playground");
  const method = playground.dataset.method;
  let pathTemplate = playground.dataset.path;
  const resultEl = playground.querySelector(".playground-result");
  const statusEl = playground.querySelector(".playground-status");
  const timeEl = playground.querySelector(".playground-time");
  const headersContentEl = playground.querySelector(".playground-response-headers-content");
  const bodyContentEl = playground.querySelector(".playground-response-body-content");

  // Server base URL
  const serverSelect = playground.querySelector(".playground-server");
  let baseUrl = serverSelect ? serverSelect.value : "";
  baseUrl = baseUrl.replace(/\/+$/, "");

  // Collect parameters
  const params = playground.querySelectorAll(".playground-param");
  const queryParts = [];

  params.forEach(function (input) {
    const name = input.dataset.name;
    const location = input.dataset.in;
    const value = input.value.trim();

    if (!value) return;

    if (location === "path") {
      pathTemplate = pathTemplate.replace("{" + name + "}", encodeURIComponent(value));
    } else if (location === "query") {
      queryParts.push(encodeURIComponent(name) + "=" + encodeURIComponent(value));
    }
  });

  let url = baseUrl + pathTemplate;
  if (queryParts.length > 0) {
    url += "?" + queryParts.join("&");
  }

  // Headers
  const headers = {};
  const headersInput = playground.querySelector(".playground-headers");
  if (headersInput && headersInput.value.trim()) {
    headersInput.value.trim().split("\n").forEach(function (line) {
      const idx = line.indexOf(":");
      if (idx > 0) {
        const key = line.substring(0, idx).trim();
        const val = line.substring(idx + 1).trim();
        if (key) headers[key] = val;
      }
    });
  }

  // Request body
  const bodyInput = playground.querySelector(".playground-body");
  let body = null;
  if (bodyInput && bodyInput.value.trim()) {
    body = bodyInput.value.trim();
    if (!headers["Content-Type"]) {
      headers["Content-Type"] = "application/json";
    }
  }

  // Show loading state
  resultEl.style.display = "block";
  statusEl.textContent = "Sending...";
  statusEl.className = "playground-status";
  timeEl.textContent = "";
  headersContentEl.textContent = "";
  bodyContentEl.textContent = "";
  btn.disabled = true;

  const fetchOptions = { method: method, headers: headers };
  if (body && method !== "GET" && method !== "HEAD") {
    fetchOptions.body = body;
  }

  const startTime = performance.now();

  fetch(url, fetchOptions)
    .then(function (response) {
      const elapsed = Math.round(performance.now() - startTime);
      timeEl.textContent = elapsed + "ms";

      // Status
      statusEl.textContent = response.status + " " + response.statusText;
      if (response.status >= 200 && response.status < 300) {
        statusEl.className = "playground-status status-success";
      } else if (response.status >= 400) {
        statusEl.className = "playground-status status-error";
      } else {
        statusEl.className = "playground-status status-info";
      }

      // Response headers
      const respHeaders = [];
      response.headers.forEach(function (value, key) {
        respHeaders.push(key + ": " + value);
      });
      headersContentEl.textContent = respHeaders.join("\n") || "(no headers exposed)";

      // Response body
      const contentType = response.headers.get("content-type") || "";
      if (contentType.includes("json")) {
        return response.text().then(function (text) {
          try {
            bodyContentEl.textContent = JSON.stringify(JSON.parse(text), null, 2);
          } catch (e) {
            bodyContentEl.textContent = text;
          }
        });
      } else {
        return response.text().then(function (text) {
          bodyContentEl.textContent = text;
        });
      }
    })
    .catch(function (err) {
      const elapsed = Math.round(performance.now() - startTime);
      timeEl.textContent = elapsed + "ms";
      statusEl.textContent = "Error";
      statusEl.className = "playground-status status-error";
      headersContentEl.textContent = "";
      bodyContentEl.textContent = err.message;
    })
    .finally(function () {
      btn.disabled = false;
    });
}

// Search with Fuse.js
(function () {
  var fuse = null;
  var debounceTimer = null;

  function loadIndex() {
    if (fuse) return;
    fetch("/search-index.json")
      .then(function (r) { return r.json(); })
      .then(function (data) {
        fuse = new Fuse(data, {
          keys: [
            { name: "title", weight: 0.4 },
            { name: "description", weight: 0.3 },
            { name: "content", weight: 0.2 },
            { name: "section", weight: 0.1 }
          ],
          threshold: 0.3,
          includeMatches: true,
          minMatchCharLength: 2,
          limit: 10
        });
      });
  }

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  function setupSearch(inputId, resultsId, overlayEl, shortcutEl) {
    var searchInput = document.getElementById(inputId);
    var searchResults = document.getElementById(resultsId);
    if (!searchInput || !searchResults) return;

    searchInput.addEventListener("focus", loadIndex);

    searchInput.addEventListener("input", function () {
      clearTimeout(debounceTimer);
      var query = searchInput.value.trim();
      if (query.length < 2 || !fuse) {
        closeSearch();
        return;
      }
      debounceTimer = setTimeout(function () {
        var results = fuse.search(query, { limit: 8 });
        renderResults(results);
      }, 150);
    });

    function renderResults(results) {
      if (results.length === 0) {
        searchResults.innerHTML = '<div class="search-empty">No results found</div>';
        openSearch();
        return;
      }
      var html = "";
      results.forEach(function (r, idx) {
        var item = r.item;
        var activeClass = idx === 0 ? " search-result-active" : "";
        html += '<a href="' + item.url + '" class="search-result-item' + activeClass + '">';
        html += '<span class="search-result-title">' + escapeHtml(item.title) + '</span>';
        html += '<span class="search-result-section">' + escapeHtml(item.section) + '</span>';
        if (item.description) {
          html += '<span class="search-result-desc">' + escapeHtml(item.description) + '</span>';
        }
        html += "</a>";
      });
      searchResults.innerHTML = html;
      openSearch();
    }

    function openSearch() {
      searchResults.style.display = "block";
      if (overlayEl) overlayEl.style.display = "block";
      if (shortcutEl) shortcutEl.style.display = "none";
    }

    function closeSearch() {
      searchResults.style.display = "none";
      if (overlayEl) overlayEl.style.display = "none";
      if (shortcutEl && document.activeElement !== searchInput) {
        shortcutEl.style.display = "";
      }
    }

    if (overlayEl) {
      overlayEl.addEventListener("click", function () {
        searchInput.value = "";
        closeSearch();
      });
    }

    searchInput.addEventListener("blur", function () {
      setTimeout(closeSearch, 200);
    });

    searchInput.addEventListener("keydown", function (e) {
      var items = searchResults.querySelectorAll(".search-result-item");
      var active = searchResults.querySelector(".search-result-active");
      var idx = Array.prototype.indexOf.call(items, active);

      if (e.key === "ArrowDown") {
        e.preventDefault();
        if (idx < items.length - 1) {
          if (active) active.classList.remove("search-result-active");
          items[idx + 1].classList.add("search-result-active");
        }
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (idx > 0) {
          if (active) active.classList.remove("search-result-active");
          items[idx - 1].classList.add("search-result-active");
        }
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (active) {
          window.location.href = active.getAttribute("href");
        }
      } else if (e.key === "Escape") {
        searchInput.value = "";
        searchInput.blur();
        closeSearch();
      }
    });

    return searchInput;
  }

  // Desktop search
  var desktopInput = setupSearch(
    "search-input", "search-results",
    document.getElementById("search-overlay"),
    document.querySelector(".search-shortcut")
  );

  // Mobile search
  setupSearch("mobile-search-input", "mobile-search-results", null, null);

  // Global "/" shortcut (focus desktop input)
  if (desktopInput) {
    var shortcutHint = document.querySelector(".search-shortcut");
    document.addEventListener("keydown", function (e) {
      if (e.key === "/" && document.activeElement.tagName !== "INPUT" && document.activeElement.tagName !== "TEXTAREA") {
        e.preventDefault();
        desktopInput.focus();
        if (shortcutHint) shortcutHint.style.display = "none";
      }
    });
  }
})();
