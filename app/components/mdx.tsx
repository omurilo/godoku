import React from "react";
import { nodeText, slugify } from "../lib/slug";
import { CodeBlock } from "./CodeBlock";

/**
 * mdxComponents overrides only the heading tags so each gets an anchor id that
 * matches the Go-generated table of contents. Every other element renders as a
 * plain tag and is styled by the .gd-prose rules in index.css.
 */

const heading = (tag: "h1" | "h2" | "h3" | "h4") =>
  function Heading({ id, children, ...props }: any) {
    const anchor = id ?? slugify(nodeText(children));
    return React.createElement(tag, { id: anchor, ...props }, children);
  };

export const mdxComponents: Record<string, React.ComponentType<any>> = {
  h1: heading("h1"),
  h2: heading("h2"),
  h3: heading("h3"),
  h4: heading("h4"),
  pre: CodeBlock,
};
