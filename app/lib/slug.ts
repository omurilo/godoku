/**
 * slugify lowercases and hyphenates a heading string for use as an anchor id.
 *
 * It MUST stay byte-for-byte compatible with slugify() in
 * internal/generator/mdx.go so the table-of-contents hrefs generated in Go line
 * up with the ids rendered on the headings here.
 */
export function slugify(input: string): string {
  const s = input.toLowerCase().trim();
  let out = "";
  let prevDash = false;
  for (const ch of s) {
    if ((ch >= "a" && ch <= "z") || (ch >= "0" && ch <= "9")) {
      out += ch;
      prevDash = false;
    } else if (out.length > 0 && !prevDash) {
      out += "-";
      prevDash = true;
    }
  }
  return out.replace(/^-+|-+$/g, "");
}

/** nodeText flattens React children into their visible text, for slugifying. */
export function nodeText(node: unknown): string {
  if (node == null || node === false || node === true) return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(nodeText).join("");
  if (typeof node === "object" && "props" in (node as Record<string, unknown>)) {
    const props = (node as { props?: { children?: unknown } }).props;
    return nodeText(props?.children);
  }
  return "";
}
