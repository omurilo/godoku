import type { ComponentType } from "react";

export interface NavLink {
  label: string;
  href?: string;
  /** Lucide icon name (e.g. "sparkles") shown next to a category label. */
  icon?: string;
  /** Nested links render as a collapsible group in the sidebar. */
  items?: NavLink[];
}

export interface TocEntry {
  level: number; // 2 | 3 | 4
  id: string;
  title: string;
}

export interface ApiParam {
  name: string;
  in: "path" | "query" | "header";
  required?: boolean;
  type?: string;
  description?: string;
}

export interface ApiResponse {
  status: string; // "200", "404", ...
  description?: string;
  /** Pretty-printed example body shown in the Response tab. */
  example?: string;
}

export interface ApiPlaygroundData {
  method: string; // GET | POST | ...
  path: string; // /users/{id}
  baseUrl?: string;
  summary?: string;
  params?: ApiParam[];
  /** Pretty-printed example request body shown in the Request tab. */
  requestExample?: string;
  responses?: ApiResponse[];
}

export interface BannerData {
  message: string;
  color?: string; // info | tip | warning | danger | ""
  dismissible?: boolean;
}

export interface BrandLogo {
  title: string;
  href?: string;
  alt?: string;
  /** Single logo, or light/dark variants swapped by theme. */
  src?: string;
  srcLight?: string;
  srcDark?: string;
}

export interface TopNavItem {
  label: string;
  href: string;
}

export interface CatalogItem {
  title: string;
  description?: string;
  href: string;
  group?: string;
}

export interface PageLink {
  title: string;
  href: string;
}

export interface AppProps {
  /** Default export of the compiled MDX module, injected by the builder. */
  content?: ComponentType<{ components?: Record<string, unknown> }>;
  title?: string;
  description?: string;
  path?: string;
  nav?: NavLink[];
  topNav?: TopNavItem[];
  toc?: TocEntry[];
  api?: ApiPlaygroundData;
  logo?: BrandLogo;
  repoUrl?: string;
  banner?: BannerData;
  /** When false, the left navigation sidebar is hidden (homepage, catalogs). */
  sidebar?: boolean;
  /** Previous/next page links shown at the foot of a doc page. */
  prev?: PageLink;
  next?: PageLink;
  /** When present, the page renders an auto-generated section catalog. */
  catalog?: CatalogItem[];
  catalogTitle?: string;
}
