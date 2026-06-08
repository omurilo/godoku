package generator

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/omurilo/godoku/internal/builder"
	"github.com/omurilo/godoku/internal/config"
	"github.com/omurilo/godoku/internal/content"
	"github.com/omurilo/godoku/internal/openapi"
)

var templatesFS embed.FS
var staticFS embed.FS
var appFS embed.FS

func SetEmbedFS(templates, static, app embed.FS) {
	templatesFS = templates
	staticFS = static
	appFS = app
}

type Generator struct {
	Config       config.Config
	RootDir      string
	OutDir       string
	NavItems     []config.NavItem
	sitemapURLs  []string
	searchIndex  []searchEntry
	llmsEntries  []llmsEntry
	hasCustomCSS bool
	hasCustomJS  bool
	appDir       string // temp dir where the embedded React shell is materialized
	mdxActive    bool   // true once the MDX/React pipeline rendered pages this build
}

type searchEntry struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Section     string `json:"section"`
	URL         string `json:"url"`
	Content     string `json:"content,omitempty"`
}

type llmsEntry struct {
	Title       string
	Description string
	URL         string
	Content     string
}

func New(cfg config.Config, rootDir string) *Generator {
	return &Generator{
		Config:  cfg,
		RootDir: rootDir,
		OutDir:  filepath.Join(rootDir, "dist"),
	}
}

func (g *Generator) Build() error {
	if err := os.MkdirAll(g.OutDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	if err := g.copyStaticAssets(); err != nil {
		return fmt.Errorf("copying static assets: %w", err)
	}

	if err := g.copyPublicAssets(); err != nil {
		return fmt.Errorf("copying public assets: %w", err)
	}

	if err := g.copyUserStatic(); err != nil {
		return fmt.Errorf("copying user static files: %w", err)
	}

	// Build filtered navigation (hide empty sections)
	g.NavItems = g.buildNavItems()

	if err := g.buildIndex(); err != nil {
		return fmt.Errorf("building index: %w", err)
	}

	apiFiles := openapi.DiscoverAPIs(g.RootDir)
	if len(apiFiles) > 0 {
		if err := g.buildAPI(apiFiles); err != nil {
			return fmt.Errorf("building API docs: %w", err)
		}
	}

	// All section content (docs/guides/tutorials, .md and .mdx alike) is rendered
	// through the React/esbuild/sobek MDX pipeline. This is a no-op (and touches
	// no network) when the project contains no section content.
	if err := g.buildMDXPages(); err != nil {
		return fmt.Errorf("building mdx pages: %w", err)
	}

	if err := g.buildSitemap(); err != nil {
		return fmt.Errorf("building sitemap: %w", err)
	}
	if err := g.buildRobotsTxt(); err != nil {
		return fmt.Errorf("building robots.txt: %w", err)
	}
	if err := g.buildSearchIndex(); err != nil {
		return fmt.Errorf("building search index: %w", err)
	}

	if err := g.build404(); err != nil {
		return fmt.Errorf("building 404 page: %w", err)
	}

	if g.Config.LLMs.LLMsTxt || g.Config.LLMs.LLMsTxtFull {
		if err := g.buildLLMsTxt(); err != nil {
			return fmt.Errorf("building llms.txt: %w", err)
		}
	}

	return nil
}

func (g *Generator) copyStaticAssets() error {
	outStaticDir := filepath.Join(g.OutDir, "static")
	if err := os.MkdirAll(outStaticDir, 0755); err != nil {
		return err
	}

	return fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := staticFS.ReadFile(path)
		if err != nil {
			return err
		}

		outPath := filepath.Join(g.OutDir, path)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(outPath, data, 0644)
	})
}

// copyPublicAssets copies the user's public/ directory verbatim into the output
// (dist/). It is additive: files there (images, CNAME, robots.txt, etc.) sit at
// the site root and can be referenced directly from Markdown/MDX. Generated
// pages may overwrite a colliding path.
func (g *Generator) copyPublicAssets() error {
	publicDir := filepath.Join(g.RootDir, "public")
	info, err := os.Stat(publicDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(publicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(publicDir, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		out := filepath.Join(g.OutDir, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(out), 0755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(out, data, 0644)
	})
}

func (g *Generator) copyUserStatic() error {
	userStaticDir := filepath.Join(g.RootDir, "static")
	if _, err := os.Stat(userStaticDir); os.IsNotExist(err) {
		return nil
	}

	outStaticDir := filepath.Join(g.OutDir, "static")

	err := filepath.WalkDir(userStaticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(userStaticDir, path)
		outPath := filepath.Join(outStaticDir, rel)

		if d.IsDir() {
			return os.MkdirAll(outPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(outPath, data, 0644)
	})
	if err != nil {
		return err
	}

	// Detect custom files
	if _, err := os.Stat(filepath.Join(userStaticDir, "custom.css")); err == nil {
		g.hasCustomCSS = true
	}
	if _, err := os.Stat(filepath.Join(userStaticDir, "custom.js")); err == nil {
		g.hasCustomJS = true
	}

	return nil
}

func (g *Generator) loadTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"isExternal": func(href string) bool {
			return strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")
		},
		"sub": func(a, b int) int { return a - b },
		"statusClass": func(code string) string {
			if strings.HasPrefix(code, "2") {
				return "2xx"
			}
			if strings.HasPrefix(code, "3") {
				return "3xx"
			}
			if strings.HasPrefix(code, "4") {
				return "4xx"
			}
			if strings.HasPrefix(code, "5") {
				return "5xx"
			}
			return "default"
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := templatesFS.ReadFile("templates/" + entry.Name())
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(entry.Name(), ".html")
		if _, err := tmpl.New(name).Parse(string(data)); err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", entry.Name(), err)
		}
	}

	return tmpl, nil
}

type pageMeta struct {
	Title       string
	Description string
	Path        string
}

type layoutData struct {
	Config          config.Config
	NavItems        []config.NavItem
	PageTitle       string
	PageDescription string
	CanonicalURL    string
	OGType          string
	HasCustomCSS    bool
	HasCustomJS     bool
	Body            template.HTML
}

func (g *Generator) renderPage(tmpl *template.Template, templateName string, data interface{}, meta pageMeta) (string, error) {
	var bodyBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&bodyBuf, templateName, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", templateName, err)
	}

	desc := meta.Description
	if desc == "" {
		desc = g.Config.Description
	}

	ogType := "article"
	if meta.Path == "/" || meta.Path == "" {
		ogType = "website"
	}

	canonical := strings.TrimRight(g.Config.URL, "/") + meta.Path

	var pageBuf bytes.Buffer
	ld := layoutData{
		Config:          g.Config,
		NavItems:        g.NavItems,
		PageTitle:       meta.Title,
		PageDescription: desc,
		CanonicalURL:    canonical,
		OGType:          ogType,
		HasCustomCSS:    g.hasCustomCSS,
		HasCustomJS:     g.hasCustomJS,
		Body:            template.HTML(bodyBuf.String()),
	}
	if err := tmpl.ExecuteTemplate(&pageBuf, "layout", ld); err != nil {
		return "", fmt.Errorf("executing layout: %w", err)
	}

	// Track page for sitemap
	g.sitemapURLs = append(g.sitemapURLs, meta.Path)

	return pageBuf.String(), nil
}

func (g *Generator) writePage(outputPath string, htmlContent string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(htmlContent), 0644)
}

func (g *Generator) buildIndex() error {
	// If redirect is set, generate redirect index.html
	if g.Config.Redirect != "" {
		redirectHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="refresh" content="0; url=%s">
<link rel="canonical" href="%s">
</head>
<body></body>
</html>`, g.Config.Redirect, g.Config.Redirect)
		return g.writePage(filepath.Join(g.OutDir, "index.html"), redirectHTML)
	}

	// content/index.md is rendered through the MDX/React pipeline (buildMDXPages),
	// not here, so it gets the same shell as every other page. Skip the default
	// homepage when it exists.
	indexMdPath := filepath.Join(g.RootDir, "content", "index.md")
	if _, err := os.Stat(indexMdPath); err == nil {
		return nil
	}
	indexMdxPath := filepath.Join(g.RootDir, "content", "index.mdx")
	if _, err := os.Stat(indexMdxPath); err == nil {
		return nil
	}

	// Otherwise, use the default homepage
	tmpl, err := g.loadTemplates()
	if err != nil {
		return err
	}
	data := struct {
		Config config.Config
	}{
		Config: g.Config,
	}
	html, err := g.renderPage(tmpl, "index", data, pageMeta{Title: "Home", Path: "/"})
	if err != nil {
		return err
	}
	return g.writePage(filepath.Join(g.OutDir, "index.html"), html)
}

func (g *Generator) buildSection(section string, contentDir string) error {
	groups, rootPages, err := content.LoadSectionGrouped(contentDir, section)
	if err != nil {
		return err
	}

	allPages := content.AllPages(groups, rootPages)

	tmpl, err := g.loadTemplates()
	if err != nil {
		return err
	}

	sectionTitle := strings.Title(section)

	// Section index page
	indexData := sectionData{
		Config:       g.Config,
		SectionTitle: sectionTitle,
		Groups:       groups,
		RootPages:    rootPages,
		AllPages:     allPages,
	}

	html, err := g.renderPage(tmpl, "section", indexData, pageMeta{Title: sectionTitle, Path: "/" + section + "/"})
	if err != nil {
		return err
	}

	sectionDir := filepath.Join(g.OutDir, section)
	if err := g.writePage(filepath.Join(sectionDir, "index.html"), html); err != nil {
		return err
	}

	// Individual pages
	for i, page := range allPages {
		var prevPage, nextPage *content.Page
		if i > 0 {
			prevPage = &allPages[i-1]
		}
		if i < len(allPages)-1 {
			nextPage = &allPages[i+1]
		}

		// Add to search index
		g.searchIndex = append(g.searchIndex, searchEntry{
			Title:       page.Title,
			Description: page.Description,
			Section:     sectionTitle,
			URL:         page.URLPath + "/",
			Content:     stripHTML(page.Content),
		})

		// Collect for llms.txt
		g.llmsEntries = append(g.llmsEntries, llmsEntry{
			Title:       page.Title,
			Description: page.Description,
			URL:         page.URLPath + "/",
			Content:     stripHTML(page.Content),
		})

		// Build edit URL
		var editURL string
		if g.Config.EditBaseURL != "" && page.SourcePath != "" {
			rel, err := filepath.Rel(g.RootDir, page.SourcePath)
			if err == nil {
				editURL = strings.TrimRight(g.Config.EditBaseURL, "/") + "/" + filepath.ToSlash(rel)
			}
		}

		pageData := sectionData{
			Config:       g.Config,
			SectionTitle: sectionTitle,
			Groups:       groups,
			RootPages:    rootPages,
			AllPages:     allPages,
			ActiveSlug:   page.Slug,
			ActivePage:   &page,
			PrevPage:     prevPage,
			NextPage:     nextPage,
			EditURL:      editURL,
		}

		html, err := g.renderPage(tmpl, "section", pageData, pageMeta{
			Title:       page.Title,
			Description: page.Description,
			Path:        page.URLPath + "/",
		})
		if err != nil {
			return err
		}

		// Pages in groups: /section/group/slug/
		// Root pages: /section/slug/
		var pagePath string
		if page.Group != "" {
			pagePath = filepath.Join(sectionDir, page.Group, page.Slug, "index.html")
		} else {
			pagePath = filepath.Join(sectionDir, page.Slug, "index.html")
		}

		if err := g.writePage(pagePath, html); err != nil {
			return err
		}
	}

	return nil
}

type sectionData struct {
	Config       config.Config
	SectionTitle string
	Groups       []content.PageGroup
	RootPages    []content.Page
	AllPages     []content.Page
	ActiveSlug   string
	ActivePage   *content.Page
	PrevPage     *content.Page
	NextPage     *content.Page
	EditURL      string
}

func (g *Generator) buildNavItems() []config.NavItem {
	sectionDirs := map[string]string{
		"/docs":      g.Config.Sections.Docs,
		"/guides":    g.Config.Sections.Guides,
		"/tutorials": g.Config.Sections.Tutorials,
	}

	var items []config.NavItem
	for _, nav := range g.Config.Navigation {
		if dir, ok := sectionDirs[nav.Path]; ok {
			contentDir := dir
			if !filepath.IsAbs(contentDir) {
				contentDir = filepath.Join(g.RootDir, contentDir)
			}
			groups, rootPages, _ := content.LoadSectionGrouped(contentDir, "")
			if len(rootPages) == 0 && len(groups) == 0 {
				continue
			}
		} else if nav.Path == "/api" {
			apiFiles := openapi.DiscoverAPIs(g.RootDir)
			if len(apiFiles) == 0 {
				continue
			}
		}
		items = append(items, nav)
	}
	return items
}

func (g *Generator) buildAPI(apiFiles []string) error {
	docs, err := openapi.LoadAllSpecs(apiFiles, g.RootDir)
	if err != nil {
		return err
	}

	appDir, err := g.materializeApp()
	if err != nil {
		return fmt.Errorf("materializing react shell: %w", err)
	}

	customCSS := ""
	if g.hasCustomCSS {
		customCSS = "/static/custom.css"
	}

	b, err := builder.New(builder.Options{
		AppDir:       appDir,
		ReactVersion: "18.3.1",
		CustomCSSURL: customCSS,
	})
	if err != nil {
		return err
	}

	logo := buildLogo(g.Config)
	topNav := make([]map[string]any, 0, len(g.NavItems))
	for _, item := range g.NavItems {
		topNav = append(topNav, map[string]any{
			"label": item.Label,
			"href":  pageURL(item.Path),
		})
	}

	apiDir := filepath.Join(g.OutDir, "api")

	// If only one API, render directly at /api/ (no catalog)
	if len(docs) == 1 {
		return g.buildSingleAPI(b, logo, topNav, docs[0], apiDir, "/api")
	}

	// Multiple APIs: render catalog at /api/ and each API at /api/{slug}/
	items := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		items = append(items, map[string]any{
			"title":       doc.Title,
			"description": doc.Description,
			"href":        pageURL("/api/" + doc.Slug),
		})
	}

	catalogProps := map[string]any{
		"logo":         logo,
		"topNav":       topNav,
		"catalog":      items,
		"catalogTitle": "API Reference",
		"title":        "API Reference",
		"description":  "Browse all available API specifications.",
	}
	if g.Config.EditBaseURL != "" {
		catalogProps["repoUrl"] = g.Config.EditBaseURL
	}
	if g.Config.Banner.Message != "" {
		catalogProps["banner"] = map[string]any{
			"message":     g.Config.Banner.Message,
			"color":       g.Config.Banner.Color,
			"dismissible": g.Config.Banner.Dismissible,
		}
	}

	if err := b.BuildPage(builder.PageInput{
		Source:      []byte(""),
		OutDir:      apiDir,
		URLPath:     "/api/",
		AssetName:   "api-catalog",
		Title:       "API Reference",
		Description: "Browse all available API specifications.",
		Props:       catalogProps,
	}); err != nil {
		return err
	}

	for _, doc := range docs {
		specDir := filepath.Join(apiDir, doc.Slug)
		basePath := "/api/" + doc.Slug
		if err := g.buildSingleAPI(b, logo, topNav, doc, specDir, basePath); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) buildSingleAPI(b *builder.Builder, logo map[string]any, topNav []map[string]any, doc *openapi.APIDoc, outDir string, basePath string) error {
	type apiEndpointSection struct {
		Endpoint        openapi.Endpoint
		CurlExample     string
		GoExample       string
		PythonExample   string
		JSExample       string
		RequestMimeType string
	}
	type apiTagSection struct {
		Tag      string
		Sections []apiEndpointSection
	}

	sections := make([]apiEndpointSection, 0, len(doc.Endpoints))
	serverURL := ""
	if len(doc.Servers) > 0 {
		serverURL = doc.Servers[0].URL
	}
	for _, endpoint := range doc.Endpoints {
		contentType := ""
		if endpoint.RequestBody != nil {
			for ct := range endpoint.RequestBody.Content {
				contentType = ct
				break
			}
		}

		sections = append(sections, apiEndpointSection{
			Endpoint:        endpoint,
			CurlExample:     APIExample(LangCurl, serverURL, endpoint, contentType),
			GoExample:       APIExample(LangGo, serverURL, endpoint, contentType),
			PythonExample:   APIExample(LangPython, serverURL, endpoint, contentType),
			JSExample:       APIExample(LangJS, serverURL, endpoint, contentType),
			RequestMimeType: contentType,
		})
	}

	sectionsByTag := make(map[string][]apiEndpointSection)
	for i, endpoint := range doc.Endpoints {
		tag := "default"
		if len(endpoint.Tags) > 0 && strings.TrimSpace(endpoint.Tags[0]) != "" {
			tag = endpoint.Tags[0]
		}
		sectionsByTag[tag] = append(sectionsByTag[tag], sections[i])
	}

	tagSections := make([]apiTagSection, 0, len(sectionsByTag))
	usedTags := make(map[string]bool)
	for _, tag := range doc.Tags {
		group, ok := sectionsByTag[tag.Name]
		if !ok || len(group) == 0 {
			continue
		}
		tagSections = append(tagSections, apiTagSection{Tag: tag.Name, Sections: group})
		usedTags[tag.Name] = true
	}

	extraTags := make([]string, 0)
	for tag := range sectionsByTag {
		if !usedTags[tag] {
			extraTags = append(extraTags, tag)
		}
	}
	sort.Strings(extraTags)
	for _, tag := range extraTags {
		tagSections = append(tagSections, apiTagSection{Tag: tag, Sections: sectionsByTag[tag]})
	}

	apiNav := make([]map[string]any, 0, len(tagSections))
	apiGroups := make([]map[string]any, 0, len(tagSections))
	for _, tag := range tagSections {
		tagSlug := slugify(tag.Tag)
		tagPath := basePath + "/" + tagSlug
		items := make([]map[string]any, 0, len(tag.Sections))
		ops := make([]map[string]any, 0, len(tag.Sections))
		for _, sec := range tag.Sections {
			ep := sec.Endpoint
			items = append(items, map[string]any{
				"label": fmt.Sprintf("[%s] %s", ep.Method, firstNonEmpty(ep.Summary, ep.Path)),
				"href":  pageURL(tagPath) + "#" + ep.Slug,
			})

			params := make([]map[string]any, 0, len(ep.Parameters))
			for _, p := range ep.Parameters {
				t := ""
				if p.Schema != nil {
					t = p.Schema.TypeString()
				}
				params = append(params, map[string]any{
					"name":        p.Name,
					"in":          p.In,
					"required":    p.Required,
					"type":        t,
					"description": p.Description,
				})
			}

			requestTypes := make([]string, 0)
			if ep.RequestBody != nil {
				for ct, media := range ep.RequestBody.Content {
					typeStr := ""
					if media.Schema != nil {
						typeStr = media.Schema.TypeString()
					}
					requestTypes = append(requestTypes, fmt.Sprintf("%s%s", ct, conditionalType(typeStr)))
				}
				sort.Strings(requestTypes)
			}

			responseRows := make([]map[string]any, 0)
			for code, resp := range ep.Responses {
				contentTypes := make([]string, 0)
				for ct, media := range resp.Content {
					typeStr := ""
					if media.Schema != nil {
						typeStr = media.Schema.TypeString()
					}
					contentTypes = append(contentTypes, fmt.Sprintf("%s%s", ct, conditionalType(typeStr)))
				}
				sort.Strings(contentTypes)
				responseRows = append(responseRows, map[string]any{
					"status":       code,
					"description":  resp.Description,
					"contentTypes": contentTypes,
					"example":      firstExampleJSON(resp.Content),
				})
			}
			sort.Slice(responseRows, func(i, j int) bool {
				return fmt.Sprint(responseRows[i]["status"]) < fmt.Sprint(responseRows[j]["status"])
			})

			requestBodyExample := ""
			if ep.RequestBody != nil {
				requestBodyExample = firstExampleJSON(ep.RequestBody.Content)
			}

			examples := map[string]string{
				"curl":   sec.CurlExample,
				"go":     sec.GoExample,
				"python": sec.PythonExample,
				"js":     sec.JSExample,
			}

			ops = append(ops, map[string]any{
				"slug":               ep.Slug,
				"method":             ep.Method,
				"path":               ep.Path,
				"summary":            firstNonEmpty(ep.Summary, ep.Path),
				"description":        ep.Description,
				"parameters":         params,
				"requestBody":        ep.RequestBody != nil,
				"requestBodyText":    firstNonEmpty(requestBodyDescription(ep), ""),
				"requestBodyExample": requestBodyExample,
				"requestBodyFields":  requestBodyFields(ep),
				"requestTypes":       requestTypes,
				"responses":          responseRows,
				"examples":           examples,
			})
		}

		apiNav = append(apiNav, map[string]any{
			"label": tag.Tag,
			"href":  pageURL(tagPath),
			"items": items,
		})
		apiGroups = append(apiGroups, map[string]any{
			"name":       tag.Tag,
			"slug":       tagSlug,
			"operations": ops,
		})
	}

	servers := make([]map[string]any, 0, len(doc.Servers))
	for _, s := range doc.Servers {
		servers = append(servers, map[string]any{"url": s.URL, "description": s.Description})
	}

	if len(apiGroups) == 0 {
		return nil
	}

	tagDesc := map[string]string{}
	for _, t := range doc.Tags {
		tagDesc[t.Name] = t.Description
	}

	baseProps := func() map[string]any {
		p := map[string]any{"logo": logo, "topNav": topNav, "nav": apiNav}
		if g.Config.EditBaseURL != "" {
			p["repoUrl"] = g.Config.EditBaseURL
		}
		if g.Config.Banner.Message != "" {
			p["banner"] = map[string]any{
				"message":     g.Config.Banner.Message,
				"color":       g.Config.Banner.Color,
				"dismissible": g.Config.Banner.Dismissible,
			}
		}
		return p
	}

	// One page per tag (Zudoku-style): /api[/<spec>]/<tag>/.
	tagSlugForEndpoint := map[string]string{}
	for _, ts := range tagSections {
		ts2 := slugify(ts.Tag)
		for _, sec := range ts.Sections {
			tagSlugForEndpoint[sec.Endpoint.Slug] = ts2
		}
	}

	firstTagSlug := ""
	for _, grp := range apiGroups {
		tagName, _ := grp["name"].(string)
		tagSlug, _ := grp["slug"].(string)
		if firstTagSlug == "" {
			firstTagSlug = tagSlug
		}
		tagPath := basePath + "/" + tagSlug
		props := baseProps()
		props["apiReference"] = map[string]any{
			"title":       doc.Title,
			"description": tagDesc[tagName],
			"version":     doc.Version,
			"basePath":    tagPath,
			"servers":     servers,
			"groups":      []map[string]any{grp},
		}
		if err := b.BuildPage(builder.PageInput{
			Source:      []byte(""),
			OutDir:      filepath.Join(outDir, tagSlug),
			URLPath:     pageURL(tagPath),
			AssetName:   "api-" + doc.Slug + "-" + tagSlug,
			Title:       tagName + " · " + doc.Title,
			Description: firstNonEmpty(tagDesc[tagName], doc.Description),
			Props:       props,
		}); err != nil {
			return err
		}
	}

	// basePath/ redirects to the first tag page.
	if err := g.writePage(filepath.Join(outDir, "index.html"), redirectPage(pageURL(basePath+"/"+firstTagSlug))); err != nil {
		return err
	}

	// Legacy per-endpoint URLs redirect to their tag page + operation anchor.
	for _, endpoint := range doc.Endpoints {
		tagSlug := tagSlugForEndpoint[endpoint.Slug]
		target := pageURL(basePath+"/"+tagSlug) + "#" + endpoint.Slug
		g.searchIndex = append(g.searchIndex, searchEntry{
			Title:       endpoint.Method + " " + endpoint.Path,
			Description: endpoint.Summary,
			Section:     "API",
			URL:         target,
			Content:     endpoint.Description,
		})
		if err := g.writePage(filepath.Join(outDir, endpoint.Slug, "index.html"), redirectPage(target)); err != nil {
			return err
		}
	}

	return nil
}

func redirectPage(target string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="0;url=%s">
  <link rel="canonical" href="%s">
  <script>location.replace(%q);</script>
  <title>Redirecting...</title>
</head>
<body>
  <p>Redirecting to <a href="%s">%s</a>...</p>
</body>
</html>`, target, target, target, target, target)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func conditionalType(typeStr string) string {
	if strings.TrimSpace(typeStr) == "" {
		return ""
	}
	return " - " + typeStr
}

// firstExampleJSON renders a pretty example JSON for a content map, preferring
// application/json. It uses a spec-provided `example`/`examples` when present,
// falling back to a generated example from the schema.
func firstExampleJSON(content map[string]openapi.MediaType) string {
	if m, ok := content["application/json"]; ok {
		if s := mediaExampleJSON(m); s != "" {
			return s
		}
	}
	keys := make([]string, 0, len(content))
	for k := range content {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s := mediaExampleJSON(content[k]); s != "" {
			return s
		}
	}
	return ""
}

// mediaExampleJSON prefers an explicit media-type example/examples, then the
// schema's generated example.
func mediaExampleJSON(m openapi.MediaType) string {
	if m.Example != nil {
		return jsonPretty(m.Example)
	}
	if len(m.Examples) > 0 {
		keys := make([]string, 0, len(m.Examples))
		for k := range m.Examples {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if m.Examples[k].Value != nil {
				return jsonPretty(m.Examples[k].Value)
			}
		}
	}
	if m.Schema != nil {
		return openapi.GenerateExampleObject(m.Schema)
	}
	return ""
}

func jsonPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// requestBodyFields extracts the top-level fields of a request body schema as
// param-like cards (name/type/required/description).
func requestBodyFields(ep openapi.Endpoint) []map[string]any {
	if ep.RequestBody == nil {
		return nil
	}
	var schema *openapi.Schema
	if m, ok := ep.RequestBody.Content["application/json"]; ok {
		schema = m.Schema
	} else {
		for _, m := range ep.RequestBody.Content {
			schema = m.Schema
			break
		}
	}
	if schema == nil {
		return nil
	}
	if schema.Type == "array" && schema.Items != nil {
		schema = schema.Items
	}
	if len(schema.Properties) == 0 {
		return nil
	}

	required := map[string]bool{}
	for _, r := range schema.Required {
		required[r] = true
	}
	names := make([]string, 0, len(schema.Properties))
	for k := range schema.Properties {
		names = append(names, k)
	}
	sort.Strings(names)

	fields := make([]map[string]any, 0, len(names))
	for _, name := range names {
		p := schema.Properties[name]
		typeStr := ""
		desc := ""
		if p != nil {
			typeStr = p.TypeString()
			desc = p.Description
		}
		fields = append(fields, map[string]any{
			"name":        name,
			"type":        typeStr,
			"required":    required[name],
			"description": desc,
		})
	}
	return fields
}

func requestBodyDescription(ep openapi.Endpoint) string {
	if ep.RequestBody == nil {
		return ""
	}
	if strings.TrimSpace(ep.RequestBody.Description) != "" {
		return ep.RequestBody.Description
	}
	if ep.RequestBody.Required {
		return "Required"
	}
	return ""
}

func (g *Generator) buildSitemap() error {
	baseURL := strings.TrimRight(g.Config.URL, "/")

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	for _, path := range g.sitemapURLs {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + baseURL + path + "</loc>\n")
		sb.WriteString("  </url>\n")
	}

	sb.WriteString("</urlset>\n")

	return g.writePage(filepath.Join(g.OutDir, "sitemap.xml"), sb.String())
}

func (g *Generator) buildRobotsTxt() error {
	baseURL := strings.TrimRight(g.Config.URL, "/")

	content := "User-agent: *\nAllow: /\n\nSitemap: " + baseURL + "/sitemap.xml\n"
	return g.writePage(filepath.Join(g.OutDir, "robots.txt"), content)
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	text := htmlTagRe.ReplaceAllString(s, " ")
	// Collapse whitespace
	parts := strings.Fields(text)
	result := strings.Join(parts, " ")
	// Truncate to keep index size reasonable
	if len(result) > 500 {
		result = result[:500]
	}
	return result
}

func (g *Generator) buildSearchIndex() error {
	data, err := json.Marshal(g.searchIndex)
	if err != nil {
		return fmt.Errorf("marshaling search index: %w", err)
	}
	return g.writePage(filepath.Join(g.OutDir, "search-index.json"), string(data))
}

func (g *Generator) build404() error {
	// When the MDX/React pipeline ran, it already rendered 404.html via the shell.
	if g.mdxActive {
		return nil
	}

	tmpl, err := g.loadTemplates()
	if err != nil {
		return err
	}

	notFoundHTML := `<div class="not-found">
	<h1>404</h1>
	<p>Page not found</p>
	<p class="not-found-desc">The page you're looking for doesn't exist or has been moved.</p>
	<a href="/" class="btn">Go Home</a>
</div>`

	html, err := g.renderPage(tmpl, "raw", struct{ Content template.HTML }{Content: template.HTML(notFoundHTML)}, pageMeta{
		Title: "Page Not Found",
		Path:  "/404.html",
	})
	if err != nil {
		// If "raw" template doesn't exist, write a simple page
		ld := layoutData{
			Config:          g.Config,
			NavItems:        g.NavItems,
			PageTitle:       "Page Not Found",
			PageDescription: "The page you're looking for doesn't exist.",
			CanonicalURL:    g.Config.URL + "/404.html",
			OGType:          "website",
			HasCustomCSS:    g.hasCustomCSS,
			HasCustomJS:     g.hasCustomJS,
			Body:            template.HTML(notFoundHTML),
		}
		var pageBuf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&pageBuf, "layout", ld); err != nil {
			return err
		}
		return g.writePage(filepath.Join(g.OutDir, "404.html"), pageBuf.String())
	}

	return g.writePage(filepath.Join(g.OutDir, "404.html"), html)
}

func (g *Generator) buildLLMsTxt() error {
	baseURL := strings.TrimRight(g.Config.URL, "/")

	if g.Config.LLMs.LLMsTxt {
		var sb strings.Builder
		sb.WriteString("# " + g.Config.Title + "\n\n")
		if g.Config.Description != "" {
			sb.WriteString("> " + g.Config.Description + "\n\n")
		}
		for _, entry := range g.llmsEntries {
			line := "- [" + entry.Title + "](" + baseURL + entry.URL + ")"
			if entry.Description != "" {
				line += ": " + entry.Description
			}
			sb.WriteString(line + "\n")
		}
		if err := g.writePage(filepath.Join(g.OutDir, "llms.txt"), sb.String()); err != nil {
			return err
		}
	}

	if g.Config.LLMs.LLMsTxtFull {
		var sb strings.Builder
		sb.WriteString("# " + g.Config.Title + "\n\n")
		if g.Config.Description != "" {
			sb.WriteString("> " + g.Config.Description + "\n\n")
		}
		for _, entry := range g.llmsEntries {
			sb.WriteString("## " + entry.Title + "\n\n")
			if entry.Content != "" {
				sb.WriteString(entry.Content + "\n\n")
			}
		}
		if err := g.writePage(filepath.Join(g.OutDir, "llms-full.txt"), sb.String()); err != nil {
			return err
		}
	}

	return nil
}
