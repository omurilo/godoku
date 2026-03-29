// Godoku - client-side enhancements
document.addEventListener("DOMContentLoaded", function () {
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
      e.preventDefault();
      const target = document.querySelector(this.getAttribute("href"));
      if (target) {
        target.scrollIntoView({ behavior: "smooth" });
      }
    });
  });

  // Collapsible sidebar groups with localStorage persistence
  var STORAGE_KEY = "godoku-collapsed-groups";

  function getCollapsedGroups() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY)) || {};
    } catch (e) {
      return {};
    }
  }

  function saveCollapsedGroups(state) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch (e) {}
  }

  var collapsedState = getCollapsedGroups();

  document.querySelectorAll(".sidebar-group").forEach(function (group) {
    var toggle = group.querySelector(".sidebar-group-toggle");
    if (!toggle) return;
    var groupName = toggle.textContent.trim();
    var hasActive = group.querySelector(".sidebar-link.active");

    // Use saved state if exists, otherwise collapse groups without active page
    var shouldCollapse;
    if (groupName in collapsedState) {
      shouldCollapse = collapsedState[groupName];
      // Always expand the group with active page regardless of saved state
      if (hasActive) shouldCollapse = false;
    } else {
      shouldCollapse = !hasActive;
    }

    if (shouldCollapse) {
      group.classList.add("collapsed");
      toggle.setAttribute("aria-expanded", "false");
    }
  });

  document.querySelectorAll(".sidebar-group-toggle").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var group = btn.closest(".sidebar-group");
      var isCollapsed = group.classList.toggle("collapsed");
      btn.setAttribute("aria-expanded", isCollapsed ? "false" : "true");

      // Persist state
      var state = getCollapsedGroups();
      state[btn.textContent.trim()] = isCollapsed;
      saveCollapsedGroups(state);
    });
  });
});

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
  var searchInput = document.getElementById("search-input");
  var searchResults = document.getElementById("search-results");
  var searchOverlay = document.getElementById("search-overlay");
  var shortcutHint = document.querySelector(".search-shortcut");
  var fuse = null;
  var debounceTimer = null;

  if (!searchInput) return;

  // Load search index lazily on first focus
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
    searchOverlay.style.display = "block";
    if (shortcutHint) shortcutHint.style.display = "none";
  }

  function closeSearch() {
    searchResults.style.display = "none";
    searchOverlay.style.display = "none";
    if (shortcutHint && document.activeElement !== searchInput) {
      shortcutHint.style.display = "";
    }
  }

  searchOverlay.addEventListener("click", function () {
    searchInput.value = "";
    closeSearch();
  });

  searchInput.addEventListener("blur", function () {
    // Delay so clicks on results register
    setTimeout(closeSearch, 200);
  });

  // Keyboard navigation
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

  // Global "/" shortcut
  document.addEventListener("keydown", function (e) {
    if (e.key === "/" && document.activeElement.tagName !== "INPUT" && document.activeElement.tagName !== "TEXTAREA") {
      e.preventDefault();
      searchInput.focus();
      if (shortcutHint) shortcutHint.style.display = "none";
    }
  });

  function escapeHtml(str) {
    var div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }
})();
