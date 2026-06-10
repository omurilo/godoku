import React from "react";
import { runtimeImport } from "../lib/cdn";

/**
 * LucideIcon renders a Lucide icon by name, loaded at runtime from esm.sh.
 * We load SVG tuple data from `lucide` (not `lucide-react`) and render it with
 * React.createElement, avoiding cross-runtime React element incompatibilities.
 */

const LUCIDE_URL = "https://esm.sh/lucide@0.460.0";

type SvgTuple = [string, Record<string, any>, Array<[string, Record<string, any>]>?];

let modPromise: Promise<any> | null = null;
function getLucide() {
  if (!modPromise) modPromise = runtimeImport(LUCIDE_URL);
  return modPromise;
}

function pascalCase(name: string): string {
  return name
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join("");
}

export function LucideIcon({ name, size = 15 }: { name: string; size?: number }) {
  const [iconDef, setIconDef] = React.useState<SvgTuple | null>(null);

  React.useEffect(() => {
    let cancelled = false;
    getLucide()
      .then((mod: any) => {
        if (cancelled) return;
        const C = mod[pascalCase(name)];
        if (Array.isArray(C) && C[0] === "svg") setIconDef(C as SvgTuple);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [name]);

  if (!iconDef) {
    return <span className="gd-icon-ph" style={{ width: size, height: size }} aria-hidden="true" />;
  }

  const [, attrs, children = []] = iconDef;
  const svgProps = mapSvgAttrs(attrs, {
    width: size,
    height: size,
    className: "gd-icon",
    "aria-hidden": "true",
  });

  return React.createElement(
    "svg",
    svgProps,
    ...children.map(([tag, childAttrs], i) => React.createElement(tag, { key: i, ...mapSvgAttrs(childAttrs) })),
  );
}

function mapSvgAttrs(attrs: Record<string, any>, extra?: Record<string, any>): Record<string, any> {
  const out: Record<string, any> = { ...attrs };
  if ("class" in out) {
    out.className = out.class;
    delete out.class;
  }
  if ("stroke-width" in out) {
    out.strokeWidth = out["stroke-width"];
    delete out["stroke-width"];
  }
  if ("stroke-linecap" in out) {
    out.strokeLinecap = out["stroke-linecap"];
    delete out["stroke-linecap"];
  }
  if ("stroke-linejoin" in out) {
    out.strokeLinejoin = out["stroke-linejoin"];
    delete out["stroke-linejoin"];
  }
  return extra ? { ...out, ...extra } : out;
}
