import React from "react";
import { cn } from "../lib/utils";
import { LucideIcon } from "./LucideIcon";
import type { NavLink } from "../types";

function isActive(href: string | undefined, current: string | undefined): boolean {
  if (!href || !current) return false;
  return href.replace(/\/$/, "") === current.replace(/\/$/, "");
}

function NavLeaf({ link, current }: { link: NavLink; current?: string }) {
  const active = isActive(link.href, current);
  const methodMatch = /^\[(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\]\s+(.+)$/i.exec(link.label || "");
  const method = methodMatch ? methodMatch[1].toUpperCase() : "";
  const label = methodMatch ? methodMatch[2] : link.label;
  return (
    <a href={link.href} aria-current={active ? "page" : undefined} className="gd-nav-link">
      <span className="gd-nav-link-label">
        {link.icon ? <LucideIcon name={link.icon} size={14} /> : null}
        {method ? <span className={cn("gd-method", "gd-method--" + method.toLowerCase(), "gd-nav-method")}>{method}</span> : null}
        <span className="gd-nav-text">{label}</span>
      </span>
    </a>
  );
}

function groupContainsActive(link: NavLink, current?: string): boolean {
  if (isActive(link.href, current)) return true;
  return (link.items ?? []).some(
    (c) => isActive(c.href, current) || (c.items ?? []).some((g) => isActive(g.href, current)),
  );
}

function NavGroup({ link, current }: { link: NavLink; current?: string }) {
  const storageKey = "godoku-nav:" + link.label;
  // Default: only the category containing the current page is expanded (keeps a
  // large sidebar compact). A user's explicit toggle is remembered in
  // localStorage and applied after mount. Initial render is deterministic
  // (based on currentPath) so SSR and hydration agree.
  const containsActive = groupContainsActive(link, current);
  const [open, setOpen] = React.useState(containsActive);

  // sessionStorage (not localStorage) so collapse state is per-session: a fresh
  // visit starts compact (only the active category open) rather than restoring
  // stale state from a previous visit.
  React.useEffect(() => {
    const stored = window.sessionStorage.getItem(storageKey);
    if (stored === "1") setOpen(true);
    else if (stored === "0") setOpen(false);
  }, [storageKey]);

  const toggle = () =>
    setOpen((o: boolean) => {
      const next = !o;
      window.sessionStorage.setItem(storageKey, next ? "1" : "0");
      return next;
    });

  // Category headers only toggle their list; the category's catalog page is
  // reachable by direct URL, not from the sidebar.
  return (
    <div className="gd-nav-group">
      <button type="button" onClick={toggle} className="gd-nav-group-btn" aria-expanded={open}>
        <span className="gd-nav-group-label">
          {link.icon ? <LucideIcon name={link.icon} /> : null}
          {link.label}
        </span>
        <ChevronIcon open={open} />
      </button>
      {open ? (
        <div className="gd-nav-group-items">
          {(link.items ?? []).map((child, i) =>
            child.items?.length ? (
              <NavGroup key={i} link={child} current={current} />
            ) : (
              <NavLeaf key={i} link={child} current={current} />
            ),
          )}
        </div>
      ) : null}
    </div>
  );
}

/** NavTree renders the nav items; shared by the desktop sidebar and mobile drawer. */
export function NavTree({ nav, currentPath }: { nav: NavLink[]; currentPath?: string }) {
  return (
    <>
      {nav.map((link, i) =>
        link.items?.length ? (
          <NavGroup key={i} link={link} current={currentPath} />
        ) : (
          <NavLeaf key={i} link={link} current={currentPath} />
        ),
      )}
    </>
  );
}

export function Sidebar({ nav, currentPath }: { nav: NavLink[]; currentPath?: string }) {
  return (
    <aside className="gd-sidebar">
      <nav className="gd-nav">
        <NavTree nav={nav} currentPath={currentPath} />
      </nav>
    </aside>
  );
}

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      className={cn("gd-chevron", open && "gd-chevron--open")}
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  );
}
