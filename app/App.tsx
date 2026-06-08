import React from "react";
import { cn } from "./lib/utils";
import { Topbar } from "./components/Topbar";
import { Sidebar } from "./components/Sidebar";
import { RightColumn } from "./components/RightColumn";
import { Catalog } from "./components/Catalog";
import { Banner } from "./components/Banner";
import { PageNav } from "./components/PageNav";
import { ApiReference } from "./components/ApiReference";
import { mdxComponents } from "./components/mdx";
import type { AppProps } from "./types";

/**
 * App is the documentation shell, rendered to static markup on the server
 * (sobek) and hydrated on the client with the same props. The tree must be
 * deterministic on first render.
 *
 * The `content` prop is the default export of the compiled MDX module; it is
 * rendered with our `mdxComponents` map in the center column.
 */
export default function App(props: AppProps) {
  const {
    content: Content,
    nav = [],
    topNav = [],
    toc = [],
    api,
    title,
    description,
    path,
    logo,
    repoUrl,
    banner,
    catalog,
    catalogTitle,
    apiReference,
  } = props;

  const isCatalog = Array.isArray(catalog) && catalog.length > 0;
  const isAPI = !!apiReference;
  const showSidebar = props.sidebar !== false && !isCatalog && nav.length > 0;
  const showRight = !isCatalog && !isAPI && showSidebar;

  return (
    <div className="gd-shell">
      {banner?.message ? <Banner banner={banner} /> : null}
      <Topbar logo={logo} repoUrl={repoUrl} topNav={topNav} currentPath={path} nav={nav} />

      <div className="gd-container">
        {showSidebar ? <Sidebar nav={nav} currentPath={path} /> : null}

        <main className={cn("gd-main", isAPI && "gd-main--api")}>
          {isAPI ? (
            <ApiReference data={apiReference!} />
          ) : (
            <article className={cn("gd-article", isCatalog && "gd-article--wide")}>
              {isCatalog ? (
                <Catalog title={catalogTitle ?? title} description={description} items={catalog!} />
              ) : (
                <>
                  <div className="gd-prose">
                    {Content ? (
                      <Content components={mdxComponents} />
                    ) : (
                      <FallbackContent title={title} description={description} />
                    )}
                  </div>
                  <PageNav prev={props.prev} next={props.next} />
                </>
              )}
            </article>
          )}
        </main>

        {showRight ? <RightColumn toc={toc} api={api} /> : null}
      </div>
    </div>
  );
}

function FallbackContent({ title, description }: { title?: string; description?: string }) {
  return (
    <div>
      <h1>{title ?? "Documentation"}</h1>
      {description ? <p>{description}</p> : null}
    </div>
  );
}
