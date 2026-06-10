import React from "react";
import { CodeBlock } from "./CodeBlock";
import type { ApiReferenceOperation } from "../types";

function parseHeaders(raw: string): Record<string, string> {
  const out: Record<string, string> = {};
  raw
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .forEach((line) => {
      const idx = line.indexOf(":");
      if (idx <= 0) return;
      out[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
    });
  return out;
}

function prettyJSON(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

interface RunResult {
  status: number;
  statusText: string;
  timeMs: number;
  headers: string;
  body: string;
}

export function TryItModal({
  op,
  servers,
  onClose,
}: {
  op: ApiReferenceOperation;
  servers: Array<{ url: string; description?: string }>;
  onClose: () => void;
}) {
  const params = op.parameters ?? [];
  const [server, setServer] = React.useState(servers[0]?.url ?? "");
  const [paramValues, setParamValues] = React.useState<Record<string, string>>({});
  const [headersText, setHeadersText] = React.useState("");
  const [body, setBody] = React.useState(op.requestBodyExample || "{\n  \n}");
  const [loading, setLoading] = React.useState(false);
  const [result, setResult] = React.useState<RunResult | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const hasBody = !!op.requestBody && op.method.toUpperCase() !== "GET" && op.method.toUpperCase() !== "HEAD";

  const run = async () => {
    if (!server) {
      setError("Select a server URL first.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      let path = op.path;
      const query = new URLSearchParams();
      const headers: Record<string, string> = parseHeaders(headersText);

      params.forEach((p) => {
        const key = p.in + ":" + p.name;
        const value = (paramValues[key] ?? "").trim();
        if (!value) return;
        if (p.in === "path") path = path.replace("{" + p.name + "}", encodeURIComponent(value));
        else if (p.in === "query") query.set(p.name, value);
        else if (p.in === "header") headers[p.name] = value;
      });

      const qs = query.toString();
      const url = server.replace(/\/+$/, "") + path + (qs ? "?" + qs : "");
      const method = op.method.toUpperCase();
      const init: RequestInit = { method, headers };
      if (hasBody) {
        if (!headers["Content-Type"] && !headers["content-type"]) headers["Content-Type"] = "application/json";
        init.body = body;
      }

      const started = performance.now();
      const response = await fetch(url, init);
      const ended = performance.now();
      const text = await response.text();
      const headerLines: string[] = [];
      response.headers.forEach((v, k) => headerLines.push(k + ": " + v));

      setResult({
        status: response.status,
        statusText: response.statusText,
        timeMs: Math.round(ended - started),
        headers: headerLines.join("\n"),
        body: prettyJSON(text),
      });
    } catch (e) {
      setResult(null);
      setError(e instanceof Error ? e.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="gd-tryit-overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="gd-tryit" role="dialog" aria-modal="true">
        <header className="gd-tryit-head">
          <div className="gd-tryit-title">
            <span className={"gd-method gd-method--" + op.method.toLowerCase()}>{op.method.toUpperCase()}</span>
            <code>{op.path}</code>
          </div>
          <button type="button" className="gd-tryit-close" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </header>

        <div className="gd-tryit-body">
          <label className="gd-tryit-field">
            <span>Server</span>
            <select className="gd-tryit-input" value={server} onChange={(e) => setServer(e.target.value)}>
              {servers.map((s) => (
                <option key={s.url} value={s.url}>
                  {s.url}
                </option>
              ))}
            </select>
          </label>

          {params.map((p) => {
            const key = p.in + ":" + p.name;
            return (
              <label key={key} className="gd-tryit-field">
                <span>
                  {p.name} <em>({p.in})</em>
                  {p.required ? <strong className="gd-tryit-req"> *</strong> : null}
                </span>
                <input
                  className="gd-tryit-input"
                  value={paramValues[key] ?? ""}
                  placeholder={p.type ?? "value"}
                  onChange={(e) => setParamValues((prev) => ({ ...prev, [key]: e.target.value }))}
                />
              </label>
            );
          })}

          {hasBody ? (
            <label className="gd-tryit-field">
              <span>Request body</span>
              <textarea
                className="gd-tryit-input gd-tryit-textarea"
                value={body}
                rows={8}
                onChange={(e) => setBody(e.target.value)}
              />
            </label>
          ) : null}

          <label className="gd-tryit-field">
            <span>Headers (one per line: Header: Value)</span>
            <textarea
              className="gd-tryit-input gd-tryit-textarea"
              value={headersText}
              rows={3}
              placeholder="Authorization: Bearer token"
              onChange={(e) => setHeadersText(e.target.value)}
            />
          </label>

          <button type="button" className="gd-btn gd-btn--primary gd-btn--sm gd-tryit-send" onClick={run} disabled={loading}>
            {loading ? "Sending…" : "Send request"}
          </button>

          {error ? <p className="gd-tryit-error">{error}</p> : null}

          {result ? (
            <div className="gd-tryit-result">
              <p className="gd-tryit-meta">
                <span className={"gd-resp-status " + statusClass(String(result.status))}>{result.status}</span>
                <span>{result.statusText}</span>
                <span className="gd-tryit-time">{result.timeMs}ms</span>
              </p>
              <h4 className="gd-op-h">Response body</h4>
              <div className="gd-force-dark gd-tryit-code">
                <CodeBlock>
                  <code className="language-json">{result.body || "(empty)"}</code>
                </CodeBlock>
              </div>
              {result.headers ? (
                <>
                  <h4 className="gd-op-h">Response headers</h4>
                  <div className="gd-force-dark gd-tryit-code">
                    <CodeBlock>
                      <code className="language-text">{result.headers}</code>
                    </CodeBlock>
                  </div>
                </>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function statusClass(status: string): string {
  const c = status.charAt(0);
  return c === "2" || c === "3" || c === "4" || c === "5" ? "gd-resp-" + c + "xx" : "gd-resp-default";
}
