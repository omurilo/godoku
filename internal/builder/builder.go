package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/grafana/sobek"

	mdxgo "github.com/omurilo/mdx-go"
)

// Options configures a Builder. AppDir is the only required field.
type Options struct {
	// AppDir holds the React shell on disk: App.tsx, entry files, components and
	// index.css. Bare imports inside it (react, framer-motion, ...) are resolved
	// from esm.sh; relative imports are resolved on disk by esbuild as usual.
	AppDir string

	// CacheDir is the on-disk cache for downloaded esm.sh modules. When empty a
	// directory under the user cache dir is used.
	CacheDir string

	// ReactVersion pins the React used for SSR and hydration. Defaults to 18.3.1.
	ReactVersion string

	// Dev produces unminified client bundles and React development builds.
	Dev bool

	// TailwindCDN, when set, injects a <script> in the page head (e.g. the
	// Tailwind v4 browser build). The default shell ships hand-written, bundled
	// CSS and needs no runtime Tailwind, so this is empty by default.
	TailwindCDN string

	// CustomCSSURL, when set, is linked last in the page head so a project's
	// static/custom.css can override the default theme tokens (e.g. to change
	// the accent color) without modifying the core.
	CustomCSSURL string
}

// Builder turns .mdx files into self-contained, server-rendered + hydrated
// HTML pages using esbuild (bundling), esm.sh (npm resolution) and goja
// (in-process SSR). It holds no per-page state and is safe to reuse.
type Builder struct {
	opts    Options
	appPath string // absolute path to App.tsx, used by the generated entrypoints
	esm     api.Plugin
}

// PageInput describes a single page to render.
type PageInput struct {
	MDXPath     string         // absolute path to the .md/.mdx source (optional if Source is set)
	Source      []byte         // inline MDX source; when non-nil, used instead of reading MDXPath
	OutDir      string         // directory to write the page + assets into
	OutFile     string         // output filename (default "index.html"); e.g. "404.html"
	URLPath     string         // canonical URL path, e.g. "/docs/intro/"
	AssetName   string         // base name for emitted assets ("intro" -> intro.js)
	Title       string         // <title> and og:title
	Heading     string         // when set, rendered as the page's H1 (from frontmatter title)
	Description string         // meta description
	Props       map[string]any // arbitrary props forwarded to <App>
}

const (
	contentImport    = "godoku:content"
	contentNamespace = "godoku-content"
)

// New constructs a Builder, validating the app directory and building the
// shared esm.sh resolver plugin up front so its cache is reused across pages.
func New(opts Options) (*Builder, error) {
	if opts.AppDir == "" {
		return nil, fmt.Errorf("builder: AppDir is required")
	}
	appDir, err := filepath.Abs(opts.AppDir)
	if err != nil {
		return nil, fmt.Errorf("builder: resolving AppDir: %w", err)
	}
	if _, err := os.Stat(appDir); err != nil {
		return nil, fmt.Errorf("builder: AppDir %q: %w", appDir, err)
	}
	appPath := filepath.Join(appDir, "App.tsx")
	if _, err := os.Stat(appPath); err != nil {
		return nil, fmt.Errorf("builder: expected App.tsx in AppDir: %w", err)
	}
	opts.AppDir = appDir

	esm, err := NewESMPlugin(ESMOptions{
		CacheDir:     opts.CacheDir,
		ReactVersion: opts.ReactVersion,
		Dev:          opts.Dev,
	})
	if err != nil {
		return nil, err
	}

	return &Builder{opts: opts, appPath: appPath, esm: esm}, nil
}

// BuildPage runs the full pipeline for one .mdx file:
//
//	read .mdx
//	  -> mdxgo.Compile (MDX -> ESM React module)
//	  -> esbuild pass 1 (bundle a server entry) -> goja (renderToStaticMarkup)
//	  -> esbuild pass 2 (bundle a client entry) -> hydration asset
//	  -> write index.html + <asset>.js (+ <asset>.css)
func (b *Builder) BuildPage(in PageInput) error {
	if in.AssetName == "" {
		return fmt.Errorf("builder: AssetName is required for %s", in.MDXPath)
	}

	var src []byte
	if in.Source != nil {
		src = in.Source
	} else {
		var rerr error
		src, rerr = os.ReadFile(in.MDXPath)
		if rerr != nil {
			return fmt.Errorf("reading mdx %s: %w", in.MDXPath, rerr)
		}
	}
	// Strip the leading YAML frontmatter block: it is metadata for the generator,
	// not MDX content. mdx-go would otherwise render it as a thematic break + text.
	src = stripFrontmatter(src)

	// The frontmatter title is the page's H1: prepend it and drop a duplicate
	// leading H1 from the body so authors don't have to repeat the title.
	if in.Heading != "" {
		src = injectHeading(src, in.Heading)
	}

	// Rewrite GoDoku Markdown extensions (admonitions) and, for plain .md
	// sources, escape literal braces so MDX does not treat them as expressions.
	escapeExpr := in.Source == nil && strings.EqualFold(filepath.Ext(in.MDXPath), ".md")
	src = preprocessMDX(src, escapeExpr)

	// 1) MDX -> ESM JS module exporting default MDXContent({ components }).
	mdxJS, err := mdxgo.Compile(src)
	if err != nil {
		return fmt.Errorf("compiling mdx %s: %w", in.MDXPath, err)
	}
	// Relative imports inside MDX resolve next to the source file; for sourced
	// (file-less) pages, resolve against the app dir.
	mdxDir := b.opts.AppDir
	if in.MDXPath != "" {
		mdxDir = filepath.Dir(in.MDXPath)
	}

	propsJSON, err := b.marshalProps(in)
	if err != nil {
		return err
	}

	// 2) Server-side render into static HTML via goja.
	ssrHTML, err := b.renderSSR(mdxJS, mdxDir, propsJSON)
	if err != nil {
		return fmt.Errorf("ssr %s: %w", in.MDXPath, err)
	}

	// 3) Client hydration bundle (JS + optional CSS).
	jsCode, cssCode, err := b.bundleClient(mdxJS, mdxDir)
	if err != nil {
		return fmt.Errorf("client bundle %s: %w", in.MDXPath, err)
	}

	// 4) Emit assets + index.html.
	if err := os.MkdirAll(in.OutDir, 0o755); err != nil {
		return fmt.Errorf("creating out dir %s: %w", in.OutDir, err)
	}

	// Asset hrefs are absolute (rooted at the page's URL path) so they resolve
	// correctly whether the page is requested with or without a trailing slash.
	// A relative "./asset.js" from "/docs/page" (no slash) would wrongly resolve
	// against "/docs/".
	assetBase := in.URLPath
	if !strings.HasPrefix(assetBase, "/") {
		assetBase = "/" + assetBase
	}
	if !strings.HasSuffix(assetBase, "/") {
		assetBase += "/"
	}

	jsName := in.AssetName + ".js"
	if err := os.WriteFile(filepath.Join(in.OutDir, jsName), []byte(jsCode), 0o644); err != nil {
		return fmt.Errorf("writing js asset: %w", err)
	}

	cssHref := ""
	if cssCode != "" {
		cssName := in.AssetName + ".css"
		if err := os.WriteFile(filepath.Join(in.OutDir, cssName), []byte(cssCode), 0o644); err != nil {
			return fmt.Errorf("writing css asset: %w", err)
		}
		cssHref = assetBase + cssName
	}

	pageHTML, err := b.renderShell(shellData{
		Title:         in.Title,
		Description:   in.Description,
		SSR:           ssrHTML,
		PropsJSON:     propsJSON,
		JSHref:        assetBase + jsName,
		CSSHref:       cssHref,
		CustomCSSHref: b.opts.CustomCSSURL,
		TailwindCDN:   b.opts.TailwindCDN,
	})
	if err != nil {
		return fmt.Errorf("rendering shell: %w", err)
	}

	outFile := in.OutFile
	if outFile == "" {
		outFile = "index.html"
	}
	if err := os.WriteFile(filepath.Join(in.OutDir, outFile), []byte(pageHTML), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outFile, err)
	}
	return nil
}

func (b *Builder) marshalProps(in PageInput) (string, error) {
	props := map[string]any{}
	for k, v := range in.Props {
		props[k] = v
	}
	// Always expose the basics the shell relies on, without clobbering caller props.
	if _, ok := props["title"]; !ok {
		props["title"] = in.Title
	}
	if _, ok := props["description"]; !ok {
		props["description"] = in.Description
	}
	if _, ok := props["path"]; !ok {
		props["path"] = in.URLPath
	}
	data, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("marshaling props: %w", err)
	}
	return string(data), nil
}

// renderSSR bundles the server entry, runs it in goja and returns the static
// markup produced by react-dom/server's renderToStaticMarkup.
func (b *Builder) renderSSR(mdxJS, mdxDir, propsJSON string) (string, error) {
	result, err := b.bundle(b.serverEntry(), mdxJS, mdxDir, api.PlatformBrowser, api.FormatIIFE, api.ES2020, false)
	if err != nil {
		return "", err
	}
	js, ok := pickOutput(result.OutputFiles, ".js")
	if !ok {
		return "", fmt.Errorf("ssr bundle produced no js output")
	}

	vm := sobek.New()
	if _, err := vm.RunString(sobekGlobals); err != nil {
		return "", fmt.Errorf("installing sobek globals: %w", err)
	}
	if _, err := vm.RunString(string(js.Contents)); err != nil {
		return "", fmt.Errorf("evaluating ssr bundle: %w", err)
	}

	fn, ok := sobek.AssertFunction(vm.Get("GODOKU_RENDER"))
	if !ok {
		return "", fmt.Errorf("GODOKU_RENDER was not defined by the ssr bundle")
	}
	out, err := fn(sobek.Undefined(), vm.ToValue(propsJSON))
	if err != nil {
		return "", fmt.Errorf("GODOKU_RENDER threw: %w", err)
	}
	return out.String(), nil
}

// bundleClient bundles the hydration entry and returns its JS and (optional) CSS.
func (b *Builder) bundleClient(mdxJS, mdxDir string) (jsCode, cssCode string, err error) {
	target := api.ES2018
	minify := !b.opts.Dev
	result, err := b.bundle(b.clientEntry(), mdxJS, mdxDir, api.PlatformBrowser, api.FormatIIFE, target, minify)
	if err != nil {
		return "", "", err
	}
	js, ok := pickOutput(result.OutputFiles, ".js")
	if !ok {
		return "", "", fmt.Errorf("client bundle produced no js output")
	}
	jsCode = string(js.Contents)
	if css, ok := pickOutput(result.OutputFiles, ".css"); ok {
		cssCode = string(css.Contents)
	}
	return jsCode, cssCode, nil
}

// bundle is the shared esbuild invocation for both passes. The entry is fed via
// stdin (resolved against AppDir); the compiled MDX is served as a virtual
// "godoku:content" module; npm packages come from esm.sh.
func (b *Builder) bundle(entry, mdxJS, mdxDir string, platform api.Platform, format api.Format, target api.Target, minify bool) (api.BuildResult, error) {
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   entry,
			ResolveDir: b.opts.AppDir,
			Sourcefile: "godoku-entry.tsx",
			Loader:     api.LoaderTSX,
		},
		Bundle:            true,
		Write:             false,
		Outdir:            b.opts.AppDir, // only used to name in-memory outputs
		Format:            format,
		Platform:          platform,
		Target:            target,
		JSX:               api.JSXTransform,
		JSXFactory:        "React.createElement",
		JSXFragment:       "React.Fragment",
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		MinifySyntax:      minify,
		LogLevel:          api.LogLevelSilent,
		Define: map[string]string{
			"process.env.NODE_ENV": strconv.Quote(b.nodeEnv()),
		},
		Plugins: []api.Plugin{
			contentPlugin(mdxJS, mdxDir),
			b.esm,
		},
	})

	if len(result.Errors) > 0 {
		msgs := api.FormatMessages(result.Errors, api.FormatMessagesOptions{
			Color: false,
			Kind:  api.ErrorMessage,
		})
		return result, fmt.Errorf("esbuild failed:\n%s", strings.Join(msgs, "\n"))
	}
	return result, nil
}

func (b *Builder) nodeEnv() string {
	if b.opts.Dev {
		return "development"
	}
	return "production"
}

// serverEntry imports the shell + compiled MDX and exposes a synchronous render
// function that goja can invoke with a JSON props string.
func (b *Builder) serverEntry() string {
	return `
import React from "react";
import { renderToString } from "react-dom/server";
import App from ` + strconv.Quote(b.appImportPath()) + `;
import MDXContent from ` + strconv.Quote(contentImport) + `;

globalThis.GODOKU_RENDER = function (propsJSON) {
  var props = JSON.parse(propsJSON);
  var element = React.createElement(App, Object.assign({}, props, { content: MDXContent }));
  // renderToString (not renderToStaticMarkup) emits the text-node boundary
  // markers hydrateRoot needs; without them React throws hydration errors
  // (#418/#425) and the tree never becomes interactive.
  return renderToString(element);
};
`
}

// clientEntry hydrates the server markup with the same component tree and props.
func (b *Builder) clientEntry() string {
	return `
import "./index.css";
import React from "react";
import { hydrateRoot } from "react-dom/client";
import App from ` + strconv.Quote(b.appImportPath()) + `;
import MDXContent from ` + strconv.Quote(contentImport) + `;

var root = document.getElementById("root");
var props = (typeof window !== "undefined" && window.__GODOKU_PROPS__) || {};
if (root) {
  hydrateRoot(root, React.createElement(App, Object.assign({}, props, { content: MDXContent })));
}
`
}

// appImportPath returns the App.tsx path with forward slashes so it is a valid
// import specifier on every platform.
func (b *Builder) appImportPath() string {
	return filepath.ToSlash(b.appPath)
}

// contentPlugin serves the compiled MDX as the virtual module "godoku:content".
// Its ResolveDir is the directory of the source .mdx so that relative imports
// inside the MDX (e.g. `import Button from "./Button.tsx"`) resolve next to it.
func contentPlugin(mdxJS, mdxDir string) api.Plugin {
	return api.Plugin{
		Name: "godoku-content",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^godoku:content$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: contentImport, Namespace: contentNamespace}, nil
				})
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: contentNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents := mdxJS
					return api.OnLoadResult{
						Contents:   &contents,
						Loader:     api.LoaderJSX,
						ResolveDir: mdxDir,
					}, nil
				})
		},
	}
}

// stripFrontmatter removes a leading YAML frontmatter block delimited by "---"
// lines. If the source does not start with "---", it is returned unchanged.
func stripFrontmatter(src []byte) []byte {
	// Tolerate a UTF-8 BOM before the opening fence.
	s := bytes.TrimPrefix(src, []byte{0xEF, 0xBB, 0xBF})
	if !strings.HasPrefix(string(s), "---") {
		return src
	}
	text := string(s)
	// The opening fence must be its own line.
	nl := strings.IndexByte(text, '\n')
	if nl < 0 || strings.TrimSpace(text[:nl]) != "---" {
		return src
	}
	rest := text[nl+1:]
	// Find the closing fence at the start of a line.
	for idx := 0; idx < len(rest); {
		end := strings.IndexByte(rest[idx:], '\n')
		var line string
		if end < 0 {
			line = rest[idx:]
		} else {
			line = rest[idx : idx+end]
		}
		if strings.TrimSpace(line) == "---" {
			if end < 0 {
				return []byte("")
			}
			return []byte(strings.TrimLeft(rest[idx+end+1:], "\n"))
		}
		if end < 0 {
			break
		}
		idx += end + 1
	}
	// No closing fence found: not real frontmatter, leave untouched.
	return src
}

func pickOutput(files []api.OutputFile, ext string) (api.OutputFile, bool) {
	for _, f := range files {
		if strings.HasSuffix(f.Path, ext) {
			return f, true
		}
	}
	return api.OutputFile{}, false
}

// sobekGlobals provides the minimal browser/Node-ish globals that React and
// react-dom/server reference. renderToStaticMarkup is synchronous, so timers
// and microtasks are safe no-ops.
const sobekGlobals = `
var global = globalThis;
var self = globalThis;
if (typeof globalThis.process === "undefined") {
  globalThis.process = { env: {}, platform: "", version: "v18.0.0", argv: [], cwd: function () { return "/"; } };
}
if (!globalThis.process.env) { globalThis.process.env = {}; }
if (typeof globalThis.console === "undefined") { globalThis.console = {}; }
["log","info","warn","error","debug","trace","group","groupEnd","table","dir","assert","count","time","timeEnd"].forEach(function (m) {
  if (typeof globalThis.console[m] !== "function") { globalThis.console[m] = function () {}; }
});
globalThis.setTimeout = globalThis.setTimeout || function () { return 0; };
globalThis.clearTimeout = globalThis.clearTimeout || function () {};
globalThis.setInterval = globalThis.setInterval || function () { return 0; };
globalThis.clearInterval = globalThis.clearInterval || function () {};
globalThis.queueMicrotask = globalThis.queueMicrotask || function (cb) { try { cb(); } catch (e) {} };
globalThis.requestAnimationFrame = globalThis.requestAnimationFrame || function () { return 0; };
globalThis.cancelAnimationFrame = globalThis.cancelAnimationFrame || function () {};

// React 18's react-dom/server (Fizz) browser build uses TextEncoder/TextDecoder.
if (typeof globalThis.TextEncoder === "undefined") {
  globalThis.TextEncoder = function TextEncoder() {};
  globalThis.TextEncoder.prototype.encoding = "utf-8";
  globalThis.TextEncoder.prototype.encode = function (input) {
    var str = String(input == null ? "" : input);
    var bytes = [];
    for (var i = 0; i < str.length; i++) {
      var c = str.charCodeAt(i);
      if (c < 0x80) {
        bytes.push(c);
      } else if (c < 0x800) {
        bytes.push(0xc0 | (c >> 6), 0x80 | (c & 0x3f));
      } else if (c >= 0xd800 && c <= 0xdbff && i + 1 < str.length) {
        var c2 = str.charCodeAt(++i);
        var cp = 0x10000 + ((c & 0x3ff) << 10) + (c2 & 0x3ff);
        bytes.push(0xf0 | (cp >> 18), 0x80 | ((cp >> 12) & 0x3f), 0x80 | ((cp >> 6) & 0x3f), 0x80 | (cp & 0x3f));
      } else {
        bytes.push(0xe0 | (c >> 12), 0x80 | ((c >> 6) & 0x3f), 0x80 | (c & 0x3f));
      }
    }
    return Uint8Array.from(bytes);
  };
}
if (typeof globalThis.TextDecoder === "undefined") {
  globalThis.TextDecoder = function TextDecoder() {};
  globalThis.TextDecoder.prototype.encoding = "utf-8";
  globalThis.TextDecoder.prototype.decode = function (buf) {
    var bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf || []);
    var out = "";
    for (var i = 0; i < bytes.length; ) {
      var c = bytes[i++];
      if (c < 0x80) {
        out += String.fromCharCode(c);
      } else if (c < 0xe0) {
        out += String.fromCharCode(((c & 0x1f) << 6) | (bytes[i++] & 0x3f));
      } else if (c < 0xf0) {
        out += String.fromCharCode(((c & 0x0f) << 12) | ((bytes[i++] & 0x3f) << 6) | (bytes[i++] & 0x3f));
      } else {
        var cp = ((c & 0x07) << 18) | ((bytes[i++] & 0x3f) << 12) | ((bytes[i++] & 0x3f) << 6) | (bytes[i++] & 0x3f);
        cp -= 0x10000;
        out += String.fromCharCode(0xd800 + (cp >> 10), 0xdc00 + (cp & 0x3ff));
      }
    }
    return out;
  };
}
`

type shellData struct {
	Title         string
	Description   string
	SSR           string // trusted React output, injected raw
	PropsJSON     string // JSON (json.Marshal escapes <,>,& by default)
	JSHref        string
	CSSHref       string
	CustomCSSHref string // optional user static/custom.css, linked last to win
	TailwindCDN   string
}

// shellTemplate is a text/template (NOT html/template): the SSR body must be
// emitted verbatim. Caller-controlled scalar fields are escaped explicitly via
// the template funcs below.
var shellTemplate = template.Must(template.New("shell").Funcs(template.FuncMap{
	"attr": html.EscapeString,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{attr .Title}}</title>
<meta name="description" content="{{attr .Description}}">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Geist:wght@300..700&family=Geist+Mono:wght@400..600&display=swap" rel="stylesheet">
{{- if .CSSHref}}
<link rel="stylesheet" href="{{.CSSHref}}">
{{- end}}
{{- if .TailwindCDN}}
<script src="{{.TailwindCDN}}"></script>
{{- end}}
{{- if .CustomCSSHref}}
<link rel="stylesheet" href="{{.CustomCSSHref}}">
{{- end}}
</head>
<body>
<div id="root">{{.SSR}}</div>
<script id="godoku-props" type="application/json">{{.PropsJSON}}</script>
<script>window.__GODOKU_PROPS__ = JSON.parse(document.getElementById("godoku-props").textContent);</script>
<script src="{{.JSHref}}" defer></script>
</body>
</html>
`))

func (b *Builder) renderShell(d shellData) (string, error) {
	var sb strings.Builder
	if err := shellTemplate.Execute(&sb, d); err != nil {
		return "", err
	}
	return sb.String(), nil
}
