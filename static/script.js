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
