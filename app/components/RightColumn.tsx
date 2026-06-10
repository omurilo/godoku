import React from "react";
import { cn } from "../lib/utils";
import { ApiPlayground } from "./ApiPlayground";
import type { ApiPlaygroundData, TocEntry } from "../types";

function Toc({ toc }: { toc: TocEntry[] }) {
  const [activeId, setActiveId] = React.useState<string>("");

  React.useEffect(() => {
    const headings = toc
      .map((t) => document.getElementById(t.id))
      .filter((el): el is HTMLElement => !!el);
    if (!headings.length) return;

    const offset = 96; // a heading counts as "current" once it reaches near the top
    const onScroll = () => {
      // If the page is scrolled to the bottom, the last heading is current even
      // if its section is too short to reach the top.
      const atBottom =
        window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 2;
      if (atBottom) {
        setActiveId(headings[headings.length - 1].id);
        return;
      }
      let current = headings[0].id;
      for (const h of headings) {
        if (h.getBoundingClientRect().top <= offset) current = h.id;
        else break;
      }
      setActiveId(current);
    };

    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    return () => {
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
    };
  }, [toc]);

  return (
    <div>
      <h4 className="gd-toc-title">On this page</h4>
      <ul className="gd-toc-list">
        {toc.map((entry) => (
          <li key={entry.id}>
            <a
              href={`#${entry.id}`}
              aria-current={activeId === entry.id ? "true" : undefined}
              className={cn(
                "gd-toc-link",
                entry.level === 3 && "gd-toc-link--3",
                entry.level === 4 && "gd-toc-link--4",
              )}
            >
              {entry.title}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function RightColumn({ toc, api }: { toc?: TocEntry[]; api?: ApiPlaygroundData }) {
  const hasToc = (toc ?? []).length > 0;
  if (!api && !hasToc) return null;

  return (
    <aside className="gd-rightcol">
      <div className="gd-rightcol-inner">
        {api ? <ApiPlayground api={api} /> : <Toc toc={toc ?? []} />}
      </div>
    </aside>
  );
}
