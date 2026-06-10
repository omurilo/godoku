import React from "react";
import { runtimeImport } from "../lib/cdn";
import { cn } from "../lib/utils";

interface CodeMeta {
  title?: string;
  highlight: number[];
  lineNumbers: boolean;
}

function parseRanges(spec: string): number[] {
  const out: number[] = [];
  spec
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean)
    .forEach((part) => {
      const m = /^(\d+)-(\d+)$/.exec(part);
      if (m) {
        for (let i = +m[1]; i <= +m[2]; i++) out.push(i);
      } else if (/^\d+$/.test(part)) {
        out.push(+part);
      }
    });
  return out;
}

/**
 * CodeBlock overrides the MDX <pre>. It renders plain code on the server (so
 * SSR and hydration match) then upgrades to Shiki-highlighted HTML on the
 * client. Shiki is loaded at runtime from esm.sh via dynamic import (so it
 * stays out of both the client and sobek SSR bundles) and emits
 * dual-theme (github-light / github-dark) tokens that follow the page theme.
 */

const SHIKI_URL = "https://esm.sh/shiki@1.24.0";

let shikiPromise: Promise<any> | null = null;
function getShiki() {
  if (!shikiPromise) shikiPromise = runtimeImport(SHIKI_URL);
  return shikiPromise;
}

const LANG_ALIASES: Record<string, string> = {
  sh: "bash",
  shell: "bash",
  zsh: "bash",
  yml: "yaml",
  "": "text",
};

function extract(children: any): { code: string; lang: string } {
  const el = Array.isArray(children) ? children[0] : children;
  const className: string = (el && el.props && el.props.className) || "";
  const match = /language-([\w-]+)/.exec(className);
  const raw = match ? match[1] : "";
  const lang = LANG_ALIASES[raw] ?? raw;
  let code = (el && el.props && el.props.children) ?? "";
  if (Array.isArray(code)) code = code.join("");
  return { code: String(code).replace(/\n$/, ""), lang };
}

export function CodeBlock({ children }: any) {
  const { code, lang } = React.useMemo(() => extract(children), [children]);
  const [html, setHtml] = React.useState<string | null>(null);
  const [copied, setCopied] = React.useState(false);
  const [meta, setMeta] = React.useState<CodeMeta>({ highlight: [], lineNumbers: false });
  const containerRef = React.useRef<HTMLDivElement>(null);
  const shikiRef = React.useRef<HTMLDivElement>(null);

  // Read code-meta (title / highlight / line numbers) from the wrapping
  // .gd-codemeta div the builder emits for fenced blocks that had a meta string.
  React.useEffect(() => {
    const wrap = containerRef.current?.closest(".gd-codemeta") as HTMLElement | null;
    if (!wrap) return;
    const ds = wrap.dataset;
    setMeta({
      title: ds.title || undefined,
      highlight: parseRanges(ds.highlight || ""),
      lineNumbers: ds.lineNumbers === "true",
    });
  }, []);

  // After Shiki renders, mark highlighted lines.
  React.useEffect(() => {
    if (!shikiRef.current || meta.highlight.length === 0) return;
    const lines = shikiRef.current.querySelectorAll(".line");
    meta.highlight.forEach((n) => lines[n - 1]?.classList.add("gd-line--hl"));
  }, [html, meta]);

  React.useEffect(() => {
    let cancelled = false;
    getShiki()
      .then(async (shiki: any) => {
        const opts = {
          themes: { light: "github-light", dark: "github-dark" },
          defaultColor: false,
        };
        let out = "";
        try {
          out = await shiki.codeToHtml(code, { lang: lang || "text", ...opts });
        } catch {
          out = await shiki.codeToHtml(code, { lang: "text", ...opts });
        }
        if (!cancelled) setHtml(out);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [code, lang]);

  const copy = () => {
    if (!navigator.clipboard) return;
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    });
  };

  return (
    <div
      ref={containerRef}
      className={cn("gd-code", meta.lineNumbers && "gd-code--ln", meta.title && "gd-code--titled")}
    >
      {meta.title ? (
        <div className="gd-code-title">{meta.title}</div>
      ) : (
        <span className="gd-code-lang">{lang || "text"}</span>
      )}
      <button type="button" className="gd-code-copy" onClick={copy} aria-label="Copy code">
        {copied ? "Copied!" : "Copy"}
      </button>
      {html !== null ? (
        <div ref={shikiRef} className="gd-shiki" dangerouslySetInnerHTML={{ __html: html }} />
      ) : (
        <pre className="gd-code-fallback">
          <code>{code}</code>
        </pre>
      )}
    </div>
  );
}
