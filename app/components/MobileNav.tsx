import React from "react";
import { createPortal } from "react-dom";
import { NavTree } from "./Sidebar";
import type { NavLink, TopNavItem } from "../types";

/**
 * MobileNav is the small-screen navigation: a hamburger button (hidden on
 * desktop via CSS) that opens a left drawer with the top navigation (Docs,
 * Guides, API reference…) plus the same nav tree as the sidebar. Closed by
 * default so SSR and hydration agree.
 */
export function MobileNav({
  nav,
  topNav = [],
  currentPath,
}: {
  nav: NavLink[];
  topNav?: TopNavItem[];
  currentPath?: string;
}) {
  const [open, setOpen] = React.useState(false);

  React.useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  React.useEffect(() => {
    document.body.style.overflow = open ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  if (!nav.length && !topNav.length) return null;

  const isActive = (href: string) =>
    !!currentPath && (currentPath === href || currentPath.replace(/\/$/, "") === href.replace(/\/$/, ""));

  return (
    <>
      <button type="button" className="gd-hamburger" aria-label="Open navigation" onClick={() => setOpen(true)}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M3 6h18M3 12h18M3 18h18" />
        </svg>
      </button>

      {open
        ? createPortal(
            <div className="gd-drawer-overlay" onMouseDown={(e) => e.target === e.currentTarget && setOpen(false)}>
              <div className="gd-drawer" role="dialog" aria-modal="true">
                <div className="gd-drawer-head">
                  <span className="gd-drawer-title">Navigation</span>
                  <button type="button" className="gd-drawer-close" aria-label="Close" onClick={() => setOpen(false)}>
                    ✕
                  </button>
                </div>
                <nav
                  className="gd-drawer-nav"
                  onClick={(e) => {
                    const t = e.target as HTMLElement;
                    if (t.closest("a")) setOpen(false);
                  }}
                >
                  {topNav.length ? (
                    <div className="gd-drawer-top">
                      {topNav.map((item) => (
                        <a
                          key={item.href}
                          href={item.href}
                          className="gd-drawer-top-link"
                          aria-current={isActive(item.href) ? "page" : undefined}
                        >
                          {item.label}
                        </a>
                      ))}
                    </div>
                  ) : null}
                  <NavTree nav={nav} currentPath={currentPath} />
                </nav>
              </div>
            </div>,
            document.body,
          )
        : null}
    </>
  );
}
