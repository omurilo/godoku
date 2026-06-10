import React from "react";
import { Search } from "./Search";
import { MobileNav } from "./MobileNav";
import type { BrandLogo, NavLink, TopNavItem } from "../types";

function useColorScheme(): [boolean, () => void] {
  const [dark, setDark] = React.useState(false);

  React.useEffect(() => {
    const root = document.documentElement;
    const stored = window.localStorage.getItem("godoku-theme");
    const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;
    const isDark = stored ? stored === "dark" : !!prefersDark;
    root.classList.toggle("dark", isDark);
    setDark(isDark);
  }, []);

  const toggle = React.useCallback(() => {
    setDark((prev: boolean) => {
      const next = !prev;
      document.documentElement.classList.toggle("dark", next);
      window.localStorage.setItem("godoku-theme", next ? "dark" : "light");
      return next;
    });
  }, []);

  return [dark, toggle];
}

export function Topbar({
  logo,
  repoUrl,
  topNav = [],
  currentPath,
  nav = [],
}: {
  logo?: BrandLogo;
  repoUrl?: string;
  topNav?: TopNavItem[];
  currentPath?: string;
  nav?: NavLink[];
}) {
  const [dark, toggle] = useColorScheme();
  const isActive = (href: string) =>
    !!currentPath && (currentPath === href || currentPath.startsWith(href));

  return (
    <header className="gd-topbar">
      <div className="gd-topbar-inner">
        <MobileNav nav={nav} topNav={topNav} currentPath={currentPath} />
        <a href={logo?.href ?? "/"} className="gd-brand">
          <BrandMark logo={logo} />
        </a>

        {topNav.length ? (
          <nav className="gd-topnav">
            {topNav.map((item) => (
              <a
                key={item.href}
                href={item.href}
                className="gd-topnav-link"
                aria-current={isActive(item.href) ? "true" : undefined}
              >
                {item.label}
              </a>
            ))}
          </nav>
        ) : null}

        <div className="gd-actions">
          <Search />

          {repoUrl ? (
            <button
              type="button"
              className="gd-iconbtn"
              aria-label="Repository"
              onClick={() => window.open(repoUrl, "_blank")}
            >
              <GithubIcon />
            </button>
          ) : null}

          <button type="button" className="gd-iconbtn" aria-label="Toggle theme" onClick={toggle}>
            {dark ? <SunIcon /> : <MoonIcon />}
          </button>
        </div>
      </div>
    </header>
  );
}

function BrandMark({ logo }: { logo?: BrandLogo }) {
  const light = logo?.srcLight || logo?.src;
  const dark = logo?.srcDark || logo?.srcLight || logo?.src;
  const alt = logo?.alt ?? logo?.title ?? "Logo";
  const width = logo?.width ? `${logo.width}px` : 'auto';
  const height = logo?.height ? `${logo.height}px` : 'auto';

  if (light || dark) {
    return (
      <>
        <img src={light} alt={alt} style={{ width, height }} className="gd-brand-logo gd-brand-logo--light" />
        <img src={dark} alt={alt} style={{ width, height }} className="gd-brand-logo gd-brand-logo--dark" />
      </>
    );
  }
  return (
    <>
      <span className="gd-brand-mark">{(logo?.title ?? "G").slice(0, 1).toUpperCase()}</span>
      <span className="gd-brand-name">{logo?.title ?? "GoDoku"}</span>
    </>
  );
}

function GithubIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 .5a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.4-1.4-1.8-1.4-1.8-1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.7-1.6-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2 0-.4-.5-1.6.2-3.2 0 0 1-.3 3.3 1.2a11.5 11.5 0 0 1 6 0C17 5 18 5.3 18 5.3c.7 1.6.2 2.8.1 3.2.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .5Z" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" />
    </svg>
  );
}

function SunIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  );
}
