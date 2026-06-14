import React from "react";
import { cn } from "../lib/utils";
import { LucideIcon } from "./LucideIcon";
import type { FooterData } from "../types";

/**
 * Footer renders the site footer from the declarative godoku.yaml `footer`
 * config: optional link columns, social icons and a copyright line. `position`
 * (left | center) controls alignment.
 */
export function Footer({ footer }: { footer: FooterData }) {
  const columns = footer.columns ?? [];
  const social = footer.social ?? [];
  const hasContent = columns.length > 0 || social.length > 0 || !!footer.copyright;
  if (!hasContent) return null;

  const centered = footer.position === "center";

  return (
    <footer className={cn("gd-footer", centered && "gd-footer--center")}>
      <div className="gd-footer-inner">
        {columns.length > 0 ? (
          <div className="gd-footer-cols">
            {columns.map((col) => (
              <div key={col.title} className="gd-footer-col">
                <h4 className="gd-footer-col-title">{col.title}</h4>
                <ul>
                  {(col.links ?? []).map((link) => (
                    <li key={link.href}>
                      <a href={link.href} className="gd-footer-link">
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        ) : null}

        <div className="gd-footer-bottom">
          {footer.copyright ? <p className="gd-footer-copy">{footer.copyright}</p> : <span />}
          {social.length > 0 ? (
            <div className="gd-footer-social">
              {social.map((s) => (
                <a
                  key={s.href}
                  href={s.href}
                  className="gd-footer-social-link"
                  aria-label={s.label || s.icon}
                  title={s.label || s.icon}
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  <LucideIcon name={s.icon} size={18} />
                </a>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </footer>
  );
}
