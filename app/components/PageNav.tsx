import React from "react";
import type { PageLink } from "../types";

/** PageNav renders previous/next links at the foot of a documentation page. */
export function PageNav({ prev, next }: { prev?: PageLink; next?: PageLink }) {
  if (!prev && !next) return null;
  return (
    <nav className="gd-pagenav">
      {prev ? (
        <a href={prev.href} className="gd-pagenav-link gd-pagenav-prev">
          <span className="gd-pagenav-dir">← Previous</span>
          <span className="gd-pagenav-title">{prev.title}</span>
        </a>
      ) : (
        <span />
      )}
      {next ? (
        <a href={next.href} className="gd-pagenav-link gd-pagenav-next">
          <span className="gd-pagenav-dir">Next →</span>
          <span className="gd-pagenav-title">{next.title}</span>
        </a>
      ) : (
        <span />
      )}
    </nav>
  );
}
