package builder

import (
	"regexp"
	"strings"
)

// preprocessMDX rewrites GoDoku's Markdown extensions into constructs that
// mdx-go understands. It runs after frontmatter stripping and before mdx-go's
// Compile, so existing `.md` documentation renders through the React pipeline
// unchanged.
//
// escapeExpr should be true for plain Markdown (`.md`) sources, where `{`/`}`
// in prose are literal text; it is false for `.mdx`, where braces are real JSX
// expressions the author intends to evaluate.
// injectHeading makes the given title the document's H1: it drops a leading
// ATX H1 already present in the body (to avoid a duplicate) and prepends
// "# <heading>". The body is otherwise untouched.
func injectHeading(src []byte, heading string) []byte {
	body := strings.TrimLeft(string(src), "\n")
	if strings.HasPrefix(body, "# ") {
		if nl := strings.IndexByte(body, '\n'); nl >= 0 {
			body = strings.TrimLeft(body[nl+1:], "\n")
		} else {
			body = ""
		}
	}
	return []byte("# " + heading + "\n\n" + body)
}

func preprocessMDX(src []byte, escapeExpr bool) []byte {
	text := string(src)
	text = wrapCodeMeta(text)
	text = convertAdmonitions(text)
	if escapeExpr {
		text = escapeBracesOutsideCode(text)
	}
	return []byte(text)
}

var codeTitleRe = regexp.MustCompile(`title="([^"]*)"`)
var codeHighlightRe = regexp.MustCompile(`\{([0-9,\-\s]+)\}`)

// wrapCodeMeta extracts fenced-code metadata (title="…", {1,3-5}, showLineNumbers)
// from the info string, strips it so mdx-go gets a clean ```lang fence, and wraps
// the block in <div className="gd-codemeta" data-…> so the React CodeBlock can
// render a title bar, line numbers and highlighted lines.
func wrapCodeMeta(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inBlock := false
	wrapped := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBlock && strings.HasPrefix(trimmed, "```") {
			inBlock = true
			info := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			lang := info
			if i := strings.IndexAny(info, " \t"); i >= 0 {
				lang = info[:i]
			}

			title := ""
			if m := codeTitleRe.FindStringSubmatch(info); m != nil {
				title = m[1]
			}
			highlight := ""
			if m := codeHighlightRe.FindStringSubmatch(info); m != nil {
				highlight = strings.ReplaceAll(m[1], " ", "")
			}
			lineNumbers := strings.Contains(info, "showLineNumbers")

			if title != "" || highlight != "" || lineNumbers {
				wrapped = true
				attrs := ""
				if title != "" {
					attrs += ` data-title="` + jsxAttr(title) + `"`
				}
				if highlight != "" {
					attrs += ` data-highlight="` + highlight + `"`
				}
				if lineNumbers {
					attrs += ` data-line-numbers="true"`
				}
				out = append(out, "", `<div className="gd-codemeta"`+attrs+`>`, "", "```"+lang)
			} else {
				wrapped = false
				out = append(out, line)
			}
			continue
		}

		if inBlock && strings.HasPrefix(trimmed, "```") {
			inBlock = false
			out = append(out, line)
			if wrapped {
				out = append(out, "", "</div>", "")
				wrapped = false
			}
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

var admonitionTitles = map[string]string{
	"note":    "Note",
	"info":    "Info",
	"tip":     "Tip",
	"warning": "Warning",
	"danger":  "Danger",
	"caution": "Caution",
}

var admonitionOpenRe = regexp.MustCompile(`^:::(note|info|tip|warning|danger|caution)\s*(?:\{title="([^"]*)"})?\s*$`)

// convertAdmonitions turns `:::type ... :::` blocks into a block-level
// <Admonition> JSX element. Blank lines around the inner body ensure mdx-go
// parses the body as Markdown.
func convertAdmonitions(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	depth := 0
	inFence := false
	fenceMarker := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Never touch content inside fenced code blocks (e.g. ```markdown
		// examples that literally show ":::info" must stay verbatim).
		if !inFence && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			inFence = true
			fenceMarker = trimmed[:3]
			out = append(out, line)
			continue
		} else if inFence {
			if strings.HasPrefix(trimmed, fenceMarker) {
				inFence = false
			}
			out = append(out, line)
			continue
		}

		if m := admonitionOpenRe.FindStringSubmatch(trimmed); m != nil {
			adType := m[1]
			title := m[2]
			if title == "" {
				title = admonitionTitles[adType]
			}
			depth++
			// Emit a lowercase <div> (esbuild treats it as a DOM element). The
			// title goes in a data attribute rendered via CSS ::before, NOT as a
			// child <p>: a child block element here makes mdx-go mis-parse a
			// Markdown list that starts the body (it would emit the list as raw
			// text). Blank lines keep the body parsed as Markdown.
			out = append(out,
				"",
				`<div className="godoku-admonition godoku-admonition-`+adType+`" data-title="`+jsxAttr(title)+`">`,
				"",
			)
			continue
		}

		if trimmed == ":::" && depth > 0 {
			depth--
			out = append(out, "", "</div>", "")
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func jsxAttr(s string) string {
	return strings.ReplaceAll(s, `"`, "&quot;")
}

// fenceRe matches fenced code blocks (``` ... ```), inline code (`...`) and
// already-existing JSX/expressions so we can skip them when escaping braces.
var codeSpanRe = regexp.MustCompile("(?s)```.*?```|``[^`]*``|`[^`\n]*`")

// escapeBracesOutsideCode escapes `{` and `}` that appear in plain Markdown
// prose so MDX does not parse them as JSX expressions. Braces inside fenced or
// inline code are preserved verbatim (mdx-go emits those as string literals).
// Existing intentional expressions in `.mdx` files typically live in code or
// JSX; this conservative pass keeps prose like `{ "json": true }` literal.
func escapeBracesOutsideCode(text string) string {
	// Split the text into code spans (kept as-is) and the gaps between them
	// (where braces are escaped).
	var b strings.Builder
	last := 0
	for _, loc := range codeSpanRe.FindAllStringIndex(text, -1) {
		b.WriteString(escapeBraces(text[last:loc[0]]))
		b.WriteString(text[loc[0]:loc[1]]) // code span, verbatim
		last = loc[1]
	}
	b.WriteString(escapeBraces(text[last:]))
	return b.String()
}

// braceReplacer rewrites literal braces to MDX string-expressions in a single
// pass: {"{"} renders the character "{" and {"}"} renders "}". A single pass is
// required so the "}" we introduce for "{" is not itself rewritten.
var braceReplacer = strings.NewReplacer("{", `{"{"}`, "}", `{"}"}`)

func escapeBraces(s string) string {
	return braceReplacer.Replace(s)
}
