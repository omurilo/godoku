import React from "react";
import { CodeBlock } from "./CodeBlock";
import { TryItModal } from "./TryItModal";
import { cn } from "../lib/utils";
import { slugify } from "../lib/slug";
import type { ApiReferenceData, ApiReferenceOperation, ApiReferenceParam } from "../types";

const LANGS = [
  { id: "curl", label: "cURL", lang: "bash" },
  { id: "go", label: "Go", lang: "go" },
  { id: "python", label: "Python", lang: "python" },
  { id: "js", label: "JavaScript", lang: "javascript" },
];

/* ---- shared language (synced to ?lang= + localStorage) ---------------- */

const LangContext = React.createContext<{ lang: string; setLang: (l: string) => void }>({
  lang: "curl",
  setLang: () => {},
});

function useLangProvider() {
  const [lang, setLangState] = React.useState("curl");
  React.useEffect(() => {
    const fromUrl = new URL(window.location.href).searchParams.get("lang");
    const fromLs = window.localStorage.getItem("godoku-api-lang");
    const initial = fromUrl || fromLs;
    if (initial && LANGS.some((l) => l.id === initial)) setLangState(initial);
  }, []);
  const setLang = React.useCallback((l: string) => {
    setLangState(l);
    try {
      window.localStorage.setItem("godoku-api-lang", l);
      const u = new URL(window.location.href);
      u.searchParams.set("lang", l);
      window.history.replaceState(null, "", u.toString());
    } catch {
      /* ignore */
    }
  }, []);
  return { lang, setLang };
}

/* ---- small pieces ----------------------------------------------------- */

function MethodBadge({ method }: { method: string }) {
  return <span className={cn("gd-method", "gd-method--" + method.toLowerCase())}>{method.toUpperCase()}</span>;
}

function RoutePath({ path, className }: { path: string; className?: string }) {
  const parts = path.split(/(\{[^}]+\})/g).filter(Boolean);
  return (
    <code className={className}>
      {parts.map((part, i) =>
        part.startsWith("{") ? (
          <span key={i} className="gd-op-route-var">
            {part}
          </span>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
    </code>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = React.useState(false);
  return (
    <button
      type="button"
      className="gd-apiref-copy"
      aria-label="Copy"
      onClick={() => navigator.clipboard?.writeText(text).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1300); })}
    >
      {copied ? "✓" : "⧉"}
    </button>
  );
}

function ParamCard({ p }: { p: ApiReferenceParam }) {
  return (
    <div className="gd-param-card">
      <div className="gd-param-card-head">
        <code className="gd-param-name">{p.name}</code>
        {p.type ? <span className="gd-param-type">{p.type}</span> : null}
        {p.required ? <span className="gd-param-req">required</span> : null}
      </div>
      {p.description ? <p className="gd-param-desc">{p.description}</p> : null}
    </div>
  );
}

function ParamSection({ title, params }: { title: string; params: ApiReferenceParam[] }) {
  if (!params.length) return null;
  return (
    <section className="gd-op-section">
      <h4 className="gd-op-h">{title}</h4>
      <div className="gd-param-list">
        {params.map((p) => (
          <ParamCard key={p.in + ":" + p.name} p={p} />
        ))}
      </div>
    </section>
  );
}

function statusClass(status: string): string {
  const c = String(status).charAt(0);
  return c === "2" || c === "3" || c === "4" || c === "5" ? "gd-resp-" + c + "xx" : "gd-resp-default";
}

/* ---- right column: code samples + examples ---------------------------- */

function DarkCode({ code, lang, collapsible = true }: { code: string; lang: string; collapsible?: boolean }) {
  const lineCount = code.split("\n").length;
  const canCollapse = collapsible && lineCount > 10;
  const [open, setOpen] = React.useState(false);
  const collapsed = canCollapse && !open;

  return (
    <div className={cn("gd-force-dark gd-sample-code", collapsed && "gd-collapsed")}>
      <CodeBlock>
        <code className={"language-" + lang}>{code}</code>
      </CodeBlock>
      {canCollapse ? (
        <button type="button" className="gd-expand-btn" onClick={() => setOpen((o) => !o)}>
          {open ? "Show less" : `Show all ${lineCount} lines`}
        </button>
      ) : null}
    </div>
  );
}

function CodeSamples({ op, onTry }: { op: ApiReferenceOperation; onTry: () => void }) {
  const { lang, setLang } = React.useContext(LangContext);
  const meta = LANGS.find((l) => l.id === lang) ?? LANGS[0];
  const examples = op.examples ?? {};
  const code = examples[lang] || examples.curl || "";
  const responseExamples = (op.responses ?? []).filter((r) => r.example);
  const [respStatus, setRespStatus] = React.useState(responseExamples[0]?.status ?? "");
  const currentResp = responseExamples.find((r) => r.status === respStatus) ?? responseExamples[0];

  return (
    <div className="gd-samples">
      <div className="gd-op-sample gd-force-dark">
        <div className="gd-op-sample-head">
          <MethodBadge method={op.method} />
          <RoutePath path={op.path} className="gd-op-route-path" />
          <button type="button" className="gd-try-btn" onClick={onTry}>
            Try it
          </button>
        </div>
        <DarkCode code={code} lang={meta.lang} />
        <div className="gd-op-sample-foot">
          <span>Show example in</span>
          <select className="gd-op-sample-select" value={lang} aria-label="Example language" onChange={(e) => setLang(e.target.value)}>
            {LANGS.map((l) => (
              <option key={l.id} value={l.id}>
                {l.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {op.requestBodyExample ? (
        <div className="gd-sample-card">
          <div className="gd-sample-card-head">Request Body Example</div>
          <DarkCode code={op.requestBodyExample} lang="json" />
        </div>
      ) : null}

      {responseExamples.length ? (
        <div className="gd-sample-card">
          <div className="gd-sample-card-head">
            <span>Response Example</span>
            {responseExamples.length > 1 ? (
              <select
                className="gd-sample-status-select"
                value={respStatus}
                aria-label="Response status"
                onChange={(e) => setRespStatus(e.target.value)}
              >
                {responseExamples.map((r) => (
                  <option key={r.status} value={r.status}>
                    {r.status}
                  </option>
                ))}
              </select>
            ) : (
              <span className={cn("gd-resp-status", statusClass(currentResp?.status ?? ""))}>{currentResp?.status}</span>
            )}
          </div>
          <DarkCode code={currentResp?.example ?? ""} lang="json" />
        </div>
      ) : null}
    </div>
  );
}

/* ---- operation -------------------------------------------------------- */

function Operation({
  op,
  baseUrl,
  servers,
}: {
  op: ApiReferenceOperation;
  baseUrl: string;
  servers: Array<{ url: string; description?: string }>;
}) {
  const [tryOpen, setTryOpen] = React.useState(false);
  const params = op.parameters ?? [];
  const pathParams = params.filter((p) => p.in === "path");
  const queryParams = params.filter((p) => p.in === "query");
  const headerParams = params.filter((p) => p.in === "header");
  const bodyFields = op.requestBodyFields ?? [];

  return (
    <section id={op.slug} className="gd-op">
      <div className="gd-op-grid">
        <div className="gd-op-docs">
          <h3 className="gd-op-title">{op.summary}</h3>
          <div className="gd-op-route">
            <MethodBadge method={op.method} />
            <RoutePath path={baseUrl.replace(/\/$/, "") + op.path} className="gd-op-route-full" />
          </div>
          {op.description ? <p className="gd-op-desc">{op.description}</p> : null}

          <ParamSection title="Path Parameters" params={pathParams} />
          <ParamSection title="Query Parameters" params={queryParams} />
          <ParamSection title="Headers" params={headerParams} />

          {bodyFields.length ? (
            <ParamSection title="Request Body" params={bodyFields} />
          ) : op.requestBody ? (
            <section className="gd-op-section">
              <h4 className="gd-op-h">Request Body</h4>
              {op.requestBodyText ? <p className="gd-param-desc">{op.requestBodyText}</p> : null}
              {(op.requestTypes ?? []).length ? <p className="gd-op-ct">{(op.requestTypes ?? []).join(", ")}</p> : null}
            </section>
          ) : null}

          {(op.responses ?? []).length ? (
            <section className="gd-op-section">
              <h4 className="gd-op-h">Responses</h4>
              <div className="gd-resp-list">
                {(op.responses ?? []).map((r) => (
                  <div key={r.status} className="gd-resp">
                    <span className={cn("gd-resp-status", statusClass(r.status))}>{r.status}</span>
                    <div className="gd-resp-body">
                      {r.description ? <span className="gd-resp-desc">{r.description}</span> : null}
                      {(r.contentTypes ?? []).length ? (
                        <code className="gd-resp-ct">{(r.contentTypes ?? []).join(", ")}</code>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ) : null}
        </div>

        <aside className="gd-op-aside">
          <CodeSamples op={op} onTry={() => setTryOpen(true)} />
        </aside>
      </div>

      {tryOpen ? <TryItModal op={op} servers={servers} onClose={() => setTryOpen(false)} /> : null}
    </section>
  );
}

/* ---- page ------------------------------------------------------------- */

export function ApiReference({ data }: { data: ApiReferenceData }) {
  const langCtx = useLangProvider();
  const baseUrl = data.servers?.[0]?.url ?? "";
  const groups = data.groups ?? [];
  // Single-group (per-tag) page: tag name is the headline, API title the eyebrow.
  const singleGroup = groups.length === 1;

  return (
    <LangContext.Provider value={langCtx}>
      <div className="gd-apiref">
        <header className="gd-apiref-head">
          <p className="gd-apiref-eyebrow">{data.title || "API Reference"}</p>
          <div className="gd-apiref-titlerow">
            <h1 className="gd-apiref-title">{singleGroup ? groups[0].name : data.title}</h1>
            {data.version ? <span className="gd-apiref-version">v{data.version}</span> : null}
          </div>
          {baseUrl ? (
            <div className="gd-apiref-endpoint">
              <span className="gd-apiref-endpoint-label">Endpoint</span>
              <code className="gd-apiref-endpoint-url">{baseUrl}</code>
              <CopyButton text={baseUrl} />
            </div>
          ) : null}
          {data.description ? <p className="gd-apiref-desc">{data.description}</p> : null}
        </header>

        {groups.map((group) => (
          <section key={group.name} className="gd-apiref-group">
            {singleGroup ? null : (
              <h2 id={slugify(group.name)} className="gd-apiref-group-title">
                {group.name}
              </h2>
            )}
            {group.operations.map((op) => (
              <Operation key={op.slug} op={op} baseUrl={baseUrl} servers={data.servers ?? []} />
            ))}
          </section>
        ))}
      </div>
    </LangContext.Provider>
  );
}
