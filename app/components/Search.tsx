import React from "react";
import { cn } from "../lib/utils";

interface SearchEntry {
  title: string;
  description?: string;
  section?: string;
  url: string;
  content?: string;
}

/**
 * Search is a client-side command-palette over /search-index.json (emitted by
 * the generator). It renders only its trigger button on the server; the dialog
 * mounts after hydration, so SSR and client markup match.
 */
export function Search() {
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [index, setIndex] = React.useState<SearchEntry[] | null>(null);
  const [active, setActive] = React.useState(0);
  const inputRef = React.useRef<HTMLInputElement>(null);

  // Global shortcuts: Cmd/Ctrl+K toggles, Escape closes.
  React.useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((o) => !o);
      } else if (e.key === "Escape") {
        setOpen(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // Lazy-load the index the first time the dialog opens, then focus the input.
  React.useEffect(() => {
    if (!open) return;
    if (index === null) {
      fetch("/search-index.json")
        .then((r) => (r.ok ? r.json() : []))
        .then((data: SearchEntry[]) => setIndex(Array.isArray(data) ? data : []))
        .catch(() => setIndex([]));
    }
    const t = setTimeout(() => inputRef.current?.focus(), 10);
    return () => clearTimeout(t);
  }, [open, index]);

  const results = React.useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!index || !needle) return [];
    const terms = needle.split(/\s+/);
    return index
      .map((entry) => {
        const haystack = `${entry.title} ${entry.description ?? ""} ${entry.content ?? ""}`.toLowerCase();
        const score = terms.reduce((acc, term) => {
          if (entry.title.toLowerCase().includes(term)) return acc + 3;
          if (haystack.includes(term)) return acc + 1;
          return acc - 5;
        }, 0);
        return { entry, score };
      })
      .filter((r) => r.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 12)
      .map((r) => r.entry);
  }, [index, query]);

  React.useEffect(() => setActive(0), [query]);

  const navigate = (url: string) => {
    setOpen(false);
    window.location.href = url;
  };

  const onInputKey = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((i) => Math.min(i + 1, Math.max(results.length - 1, 0)));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter" && results[active]) {
      e.preventDefault();
      navigate(results[active].url);
    }
  };

  return (
    <>
      <button type="button" onClick={() => setOpen(true)} aria-label="Search" className="gd-search-trigger">
        <SearchIcon />
        <span className="gd-search-trigger-label">Search the docs</span>
        <kbd className="gd-kbd">⌘K</kbd>
      </button>

      {open ? (
        <div
          className="gd-search-overlay"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setOpen(false);
          }}
        >
          <div className="gd-search-panel" role="dialog" aria-modal="true">
            <div className="gd-search-head">
              <SearchIcon />
              <input
                ref={inputRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={onInputKey}
                placeholder="Search documentation…"
                className="gd-search-input"
              />
              <kbd className="gd-kbd">ESC</kbd>
            </div>

            <div className="gd-search-results">
              {index === null ? (
                <p className="gd-search-empty">Loading…</p>
              ) : query.trim() === "" ? (
                <p className="gd-search-empty">Type to search the documentation.</p>
              ) : results.length === 0 ? (
                <p className="gd-search-empty">No results for “{query}”.</p>
              ) : (
                <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
                  {results.map((r, i) => (
                    <li key={r.url}>
                      <a
                        href={r.url}
                        onMouseEnter={() => setActive(i)}
                        onClick={(e) => {
                          e.preventDefault();
                          navigate(r.url);
                        }}
                        className={cn("gd-search-item", i === active && "gd-search-item--active")}
                      >
                        <div className="gd-search-item-top">
                          <span className="gd-search-item-title">{r.title}</span>
                          {r.section ? <span className="gd-search-item-section">{r.section}</span> : null}
                        </div>
                        {r.description ? <p className="gd-search-item-desc">{r.description}</p> : null}
                      </a>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}

function SearchIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0 }}>
      <circle cx="11" cy="11" r="7" />
      <path d="m21 21-4.3-4.3" />
    </svg>
  );
}
