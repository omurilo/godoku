import React from "react";
import type { CatalogItem } from "../types";

/**
 * Catalog renders a section landing page (/docs/, /guides/) as a grid of cards,
 * one per document (title + short description), grouped by optional group label.
 */
export function Catalog({
  title,
  description,
  items,
}: {
  title?: string;
  description?: string;
  items: CatalogItem[];
}) {
  const groups = groupItems(items);

  return (
    <div>
      <header className="gd-catalog-header">
        <h1>{title ?? "Documentation"}</h1>
        {description ? <p>{description}</p> : null}
      </header>

      {groups.map((group) => (
        <section key={group.label || "_"} className="gd-catalog-section">
          {group.label ? <h2 className="gd-catalog-group">{group.label}</h2> : null}
          <div className="gd-catalog-grid">
            {group.items.map((item) => (
              <a key={item.href} href={item.href} className="gd-card">
                <div className="gd-card-top">
                  <h3 className="gd-card-title">{item.title}</h3>
                  <span className="gd-card-arrow">→</span>
                </div>
                {item.description ? <p className="gd-card-desc">{item.description}</p> : null}
              </a>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

function groupItems(items: CatalogItem[]): { label: string; items: CatalogItem[] }[] {
  const order: string[] = [];
  const byGroup = new Map<string, CatalogItem[]>();
  for (const item of items) {
    const key = item.group ?? "";
    if (!byGroup.has(key)) {
      byGroup.set(key, []);
      order.push(key);
    }
    byGroup.get(key)!.push(item);
  }
  return order.map((label) => ({ label, items: byGroup.get(label)! }));
}
