package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// esmNamespace is the esbuild namespace used for every module that is resolved
// to an https://esm.sh/... URL. Modules in this namespace are never read from
// disk; they are fetched over HTTP (and cached) by the OnLoad callback below.
const esmNamespace = "godoku-esm"

// ESMOptions configures the esm.sh resolver plugin.
type ESMOptions struct {
	// CDN is the base URL used to resolve bare specifiers. Defaults to
	// "https://esm.sh".
	CDN string

	// CacheDir is the directory used to persist downloaded modules between
	// builds. If empty, a directory under os.UserCacheDir is used. The cache is
	// keyed by the full request URL, so pinned/versioned URLs are immutable and
	// safe to cache forever.
	CacheDir string

	// ReactVersion pins the React major/minor used both for "react"/"react-dom"
	// themselves and for the ?external= dedupe of every other package. Defaults
	// to "18.3.1".
	ReactVersion string

	// Target is forwarded to esm.sh as ?target= so the CDN ships syntax our
	// downstream esbuild pass can consume. Defaults to "es2020".
	Target string

	// Dev appends ?dev to package URLs, pulling React's development builds with
	// warnings. Leave false for production output.
	Dev bool

	// HTTPTimeout bounds each individual module download. Defaults to 30s.
	HTTPTimeout time.Duration
}

func (o *ESMOptions) applyDefaults() error {
	if o.CDN == "" {
		o.CDN = "https://esm.sh"
	}
	o.CDN = strings.TrimRight(o.CDN, "/")
	if o.ReactVersion == "" {
		o.ReactVersion = "18.3.1"
	}
	if o.Target == "" {
		o.Target = "es2020"
	}
	if o.HTTPTimeout == 0 {
		o.HTTPTimeout = 30 * time.Second
	}
	if o.CacheDir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("resolving user cache dir: %w", err)
		}
		o.CacheDir = filepath.Join(base, "godoku", "esm")
	}
	return nil
}

// esmResolver carries the shared state (HTTP client, on-disk cache, in-memory
// memoization) used by the plugin callbacks.
type esmResolver struct {
	opts   ESMOptions
	client *http.Client

	mu  sync.Mutex
	mem map[string][]byte
}

// NewESMPlugin builds an esbuild plugin that turns the build into a Deno-style
// "import from the network" resolver: every bare import (e.g. "framer-motion")
// is rewritten to an esm.sh URL, fetched over HTTP, cached on disk, and handed
// back to esbuild as a normal module. Sub-imports emitted by esm.sh (absolute
// "/...", full "https://..." or — when ?external is used — bare "react") are
// resolved recursively against the importing URL.
func NewESMPlugin(opts ESMOptions) (api.Plugin, error) {
	if err := opts.applyDefaults(); err != nil {
		return api.Plugin{}, err
	}
	if err := os.MkdirAll(opts.CacheDir, 0o755); err != nil {
		return api.Plugin{}, fmt.Errorf("creating esm cache dir: %w", err)
	}

	r := &esmResolver{
		opts:   opts,
		client: &http.Client{Timeout: opts.HTTPTimeout},
		mem:    make(map[string][]byte),
	}

	return api.Plugin{
		Name:  "godoku-esm",
		Setup: r.setup,
	}, nil
}

func (r *esmResolver) setup(build api.PluginBuild) {
	// 1) Resolve bare/url imports originating from on-disk modules. This covers
	//    both the "file" namespace (the app's source files and virtual stdin
	//    entry) and the "godoku-content" namespace (the compiled MDX module).
	//    Relative/absolute *filesystem* paths are left untouched so esbuild's
	//    default resolver keeps ownership (resolving them against the loader's
	//    ResolveDir); everything else is treated as a bare package spec.
	fromHost := func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		p := args.Path

		// Already a network module.
		if isHTTPURL(p) {
			return api.OnResolveResult{Path: p, Namespace: esmNamespace}, nil
		}
		// Relative or absolute filesystem import: not ours.
		if strings.HasPrefix(p, ".") || strings.HasPrefix(p, "/") || filepath.IsAbs(p) {
			return api.OnResolveResult{}, nil
		}
		// Node builtins have no browser equivalent here; mark external so the
		// build fails loudly at runtime rather than 404-ing on the CDN.
		if strings.HasPrefix(p, "node:") {
			return api.OnResolveResult{Path: p, External: true}, nil
		}
		// Bare package specifier -> esm.sh URL.
		return api.OnResolveResult{Path: r.specToURL(p), Namespace: esmNamespace}, nil
	}
	build.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: "file"}, fromHost)
	build.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: contentNamespace}, fromHost)

	// 2) Resolve imports that appear *inside* an already-fetched esm.sh module.
	//    The importer is itself a URL, so relative/absolute specs resolve against
	//    it; bare specs (left behind by ?external=) become fresh esm.sh URLs.
	build.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: esmNamespace},
		func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			resolved, err := r.resolveAgainst(args.Importer, args.Path)
			if err != nil {
				return api.OnResolveResult{}, err
			}
			return api.OnResolveResult{Path: resolved, Namespace: esmNamespace}, nil
		})

	// 3) Load any module in the esm namespace by fetching it over HTTP.
	build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: esmNamespace},
		func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			body, err := r.fetch(args.Path)
			if err != nil {
				return api.OnLoadResult{}, err
			}
			contents := string(body)
			loader := loaderForURL(args.Path)
			return api.OnLoadResult{
				Contents: &contents,
				Loader:   loader,
			}, nil
		})
}

// specToURL turns a bare package specifier into an esm.sh URL. React itself is
// pinned to the configured version; every other package is asked to treat
// react/react-dom as external so the whole graph shares our single React copy
// (avoiding the classic "invalid hook call / two Reacts" failure).
func (r *esmResolver) specToURL(spec string) string {
	q := url.Values{}
	q.Set("target", r.opts.Target)
	if r.opts.Dev {
		q.Set("dev", "true")
	}

	base := r.opts.CDN + "/" + spec

	if isReactPackage(spec) {
		// Pin react & friends to one version. Adding the version to the spec only
		// when it is not already pinned by the caller.
		if !strings.Contains(spec, "@") || strings.HasPrefix(spec, "@") && strings.Count(spec, "@") < 2 {
			base = r.opts.CDN + "/" + pinReact(spec, r.opts.ReactVersion)
		}
	} else {
		q.Set("external", "react,react-dom")
	}

	return base + "?" + q.Encode()
}

// resolveAgainst resolves a child import path relative to the URL of the module
// that imported it.
func (r *esmResolver) resolveAgainst(importer, spec string) (string, error) {
	if isHTTPURL(spec) {
		return spec, nil
	}
	// Bare spec encountered inside a network module (left external by esm.sh).
	if !strings.HasPrefix(spec, ".") && !strings.HasPrefix(spec, "/") {
		return r.specToURL(spec), nil
	}
	base, err := url.Parse(importer)
	if err != nil {
		return "", fmt.Errorf("parsing importer URL %q: %w", importer, err)
	}
	ref, err := url.Parse(spec)
	if err != nil {
		return "", fmt.Errorf("parsing import %q: %w", spec, err)
	}
	return base.ResolveReference(ref).String(), nil
}

// fetch returns the bytes for a URL, consulting the in-memory map first, then
// the on-disk cache, then the network. Versioned esm.sh URLs are immutable, so
// a successful download is cached permanently.
func (r *esmResolver) fetch(rawURL string) ([]byte, error) {
	r.mu.Lock()
	if b, ok := r.mem[rawURL]; ok {
		r.mu.Unlock()
		return b, nil
	}
	r.mu.Unlock()

	cachePath := r.cachePath(rawURL)
	if b, err := os.ReadFile(cachePath); err == nil {
		r.remember(rawURL, b)
		return b, nil
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", rawURL, err)
	}
	// esm.sh varies its output on the User-Agent (it ships legacy bundles to old
	// browsers). Pose as a modern browser to get standard ESM.
	req.Header.Set("User-Agent", "Mozilla/5.0 (godoku esbuild esm-loader)")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body of %s: %w", rawURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview := string(body)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("esm.sh returned %d for %s: %s", resp.StatusCode, rawURL, preview)
	}

	if err := os.WriteFile(cachePath, body, 0o644); err != nil {
		return nil, fmt.Errorf("caching %s: %w", rawURL, err)
	}
	r.remember(rawURL, body)
	return body, nil
}

func (r *esmResolver) remember(key string, b []byte) {
	r.mu.Lock()
	r.mem[key] = b
	r.mu.Unlock()
}

func (r *esmResolver) cachePath(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(r.opts.CacheDir, hex.EncodeToString(sum[:])+ext(rawURL))
}

// --- helpers -------------------------------------------------------------

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func isReactPackage(spec string) bool {
	switch {
	case spec == "react", spec == "react-dom":
		return true
	case strings.HasPrefix(spec, "react/"),
		strings.HasPrefix(spec, "react-dom/"):
		return true
	}
	return false
}

// pinReact rewrites "react", "react-dom", "react-dom/client" etc. so the
// version sits right after the package name: "react-dom/client" with version
// 18.3.1 becomes "react-dom@18.3.1/client".
func pinReact(spec, version string) string {
	pkg := spec
	sub := ""
	if i := strings.Index(spec, "/"); i >= 0 {
		pkg = spec[:i]
		sub = spec[i:]
	}
	return pkg + "@" + version + sub
}

// loaderForURL picks an esbuild loader from the URL's path extension. esm.sh
// serves JS for extensionless package roots and *.mjs/*.js for sub-modules,
// with the occasional *.css or *.json.
func loaderForURL(rawURL string) api.Loader {
	switch strings.ToLower(ext(rawURL)) {
	case ".css":
		return api.LoaderCSS
	case ".json":
		return api.LoaderJSON
	default:
		return api.LoaderJS
	}
}

// ext returns the file extension of a URL's path, ignoring any query string.
func ext(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Ext(u.Path)
}
