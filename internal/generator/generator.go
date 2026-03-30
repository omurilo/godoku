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
	"strings"

	"github.com/omurilo/godoku/internal/config"
	"github.com/omurilo/godoku/internal/content"
	"github.com/omurilo/godoku/internal/openapi"
)

var templatesFS embed.FS
var staticFS embed.FS

func SetEmbedFS(templates, static embed.FS) {
	templatesFS = templates
	staticFS = static
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
		OutDir:  filepath.Join(rootDir, "public"),
	}
}

func (g *Generator) Build() error {
	if err := os.MkdirAll(g.OutDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	if err := g.copyStaticAssets(); err != nil {
		return fmt.Errorf("copying static assets: %w", err)
	}

	if err := g.copyUserStatic(); err != nil {
		return fmt.Errorf("copying user static files: %w", err)
	}

	// Build filtered navigation (hide empty sections)
	g.NavItems = g.buildNavItems()

	if err := g.buildIndex(); err != nil {
		return fmt.Errorf("building index: %w", err)
	}

	// --- Custom: Process root-level markdown files in content/ ---
	rootContentDir := filepath.Join(g.RootDir, "content")
	entries, err := os.ReadDir(rootContentDir)
	if err == nil {
		tmpl, tmplErr := g.loadTemplates()
		if tmplErr != nil {
			return fmt.Errorf("loading templates for root content: %w", tmplErr)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "_index.md" {
				continue
			}
			page, perr := content.ParseMarkdownFile(filepath.Join(rootContentDir, entry.Name()))
			if perr != nil || page.Draft {
				continue
			}
			// Determine output path
			var outPath, urlPath string
			if entry.Name() == "index.md" {
				outPath = filepath.Join(g.OutDir, "index.html")
				urlPath = "/"
			} else {
				outPath = filepath.Join(g.OutDir, page.Slug, "index.html")
				urlPath = "/" + page.Slug + "/"
			}
			html, rerr := g.renderPage(tmpl, "section", struct {
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
			}{
				Config:       g.Config,
				SectionTitle: "",
				Groups:       nil,
				RootPages:    []content.Page{page},
				AllPages:     []content.Page{page},
				ActiveSlug:   page.Slug,
				ActivePage:   &page,
				PrevPage:     nil,
				NextPage:     nil,
				EditURL:      "",
			}, pageMeta{
				Title:       page.Title,
				Description: page.Description,
				Path:        urlPath,
			})
			if rerr != nil {
				return fmt.Errorf("rendering root content page %s: %w", entry.Name(), rerr)
			}
			if werr := g.writePage(outPath, html); werr != nil {
				return fmt.Errorf("writing root content page %s: %w", entry.Name(), werr)
			}
			// Add to sitemap and search index
			g.sitemapURLs = append(g.sitemapURLs, urlPath)
			g.searchIndex = append(g.searchIndex, searchEntry{
				Title:       page.Title,
				Description: page.Description,
				Section:     "",
				URL:         urlPath,
				Content:     stripHTML(page.Content),
			})
		}
	}

	sections := map[string]string{
		"docs":      g.Config.Sections.Docs,
		"guides":    g.Config.Sections.Guides,
		"tutorials": g.Config.Sections.Tutorials,
	}

	for section, dir := range sections {
		contentDir := dir
		if !filepath.IsAbs(contentDir) {
			contentDir = filepath.Join(g.RootDir, contentDir)
		}
		if err := g.buildSection(section, contentDir); err != nil {
			return fmt.Errorf("building section %s: %w", section, err)
		}
	}

	apiFiles := openapi.DiscoverAPIs(g.RootDir)
	if len(apiFiles) > 0 {
		if err := g.buildAPI(apiFiles); err != nil {
			return fmt.Errorf("building API docs: %w", err)
		}
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

	// If content/index.md exists, render it as /index.html
	indexMdPath := filepath.Join(g.RootDir, "content", "index.md")
	if _, err := os.Stat(indexMdPath); err == nil {
		page, perr := content.ParseMarkdownFile(indexMdPath)
		if perr == nil && !page.Draft {
			tmpl, tmplErr := g.loadTemplates()
			if tmplErr != nil {
				return tmplErr
			}
			html, rerr := g.renderPage(tmpl, "section", struct {
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
			}{
				Config:       g.Config,
				SectionTitle: "",
				Groups:       nil,
				RootPages:    []content.Page{page},
				AllPages:     []content.Page{page},
				ActiveSlug:   page.Slug,
				ActivePage:   &page,
				PrevPage:     nil,
				NextPage:     nil,
				EditURL:      "",
			}, pageMeta{
				Title:       page.Title,
				Description: page.Description,
				Path:        "/",
			})
			if rerr != nil {
				return rerr
			}
			return g.writePage(filepath.Join(g.OutDir, "index.html"), html)
		}
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

	tmpl, err := g.loadTemplates()
	if err != nil {
		return err
	}

	apiDir := filepath.Join(g.OutDir, "api")

	// If only one API, render directly at /api/ (no catalog)
	if len(docs) == 1 {
		return g.buildSingleAPI(tmpl, docs[0], apiDir, "/api")
	}

	// Multiple APIs: render catalog at /api/ and each API at /api/{slug}/
	catalogData := struct {
		Config config.Config
		APIs   []*openapi.APIDoc
	}{
		Config: g.Config,
		APIs:   docs,
	}

	html, err := g.renderPage(tmpl, "api_catalog", catalogData, pageMeta{Title: "API Reference", Path: "/api/"})
	if err != nil {
		return err
	}
	if err := g.writePage(filepath.Join(apiDir, "index.html"), html); err != nil {
		return err
	}

	for _, doc := range docs {
		specDir := filepath.Join(apiDir, doc.Slug)
		basePath := "/api/" + doc.Slug
		if err := g.buildSingleAPI(tmpl, doc, specDir, basePath); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) buildSingleAPI(tmpl *template.Template, doc *openapi.APIDoc, outDir string, basePath string) error {
	indexData := struct {
		Config      config.Config
		Title       string
		Description string
		Version     string
		Servers     []openapi.Server
		Tags        []openapi.Tag
		Endpoints   []openapi.Endpoint
		TagGroups   map[string][]openapi.Endpoint
		ActiveSlug  string
		BasePath    string
	}{
		Config:      g.Config,
		Title:       doc.Title,
		Description: doc.Description,
		Version:     doc.Version,
		Servers:     doc.Servers,
		Tags:        doc.Tags,
		Endpoints:   doc.Endpoints,
		TagGroups:   doc.TagGroups,
		BasePath:    basePath,
	}

	html, err := g.renderPage(tmpl, "api_index", indexData, pageMeta{
		Title:       doc.Title,
		Description: doc.Description,
		Path:        basePath + "/",
	})
	if err != nil {
		return err
	}

	if err := g.writePage(filepath.Join(outDir, "index.html"), html); err != nil {
		return err
	}

	for _, endpoint := range doc.Endpoints {
		// Add to search index
		g.searchIndex = append(g.searchIndex, searchEntry{
			Title:       endpoint.Method + " " + endpoint.Path,
			Description: endpoint.Summary,
			Section:     "API",
			URL:         basePath + "/" + endpoint.Slug + "/",
			Content:     endpoint.Description,
		})

		serverURL := ""
		if len(doc.Servers) > 0 {
			serverURL = doc.Servers[0].URL
		}

		contentType := ""
		if endpoint.RequestBody != nil {
			for ct := range endpoint.RequestBody.Content {
				contentType = ct
				break
			}
		}

		curlExample := APIExample(LangCurl, serverURL, endpoint, contentType)
		goExample := APIExample(LangGo, serverURL, endpoint, contentType)
		pythonExample := APIExample(LangPython, serverURL, endpoint, contentType)
		jsExample := APIExample(LangJS, serverURL, endpoint, contentType)

		epData := struct {
			Config        config.Config
			Endpoint      openapi.Endpoint
			TagGroups     map[string][]openapi.Endpoint
			ActiveSlug    string
			BasePath      string
			Servers       []openapi.Server
			CurlExample   string
			GoExample     string
			PythonExample string
			JSExample     string
		}{
			Config:        g.Config,
			Endpoint:      endpoint,
			TagGroups:     doc.TagGroups,
			ActiveSlug:    endpoint.Slug,
			BasePath:      basePath,
			Servers:       doc.Servers,
			CurlExample:   curlExample,
			GoExample:     goExample,
			PythonExample: pythonExample,
			JSExample:     jsExample,
		}

		html, err := g.renderPage(tmpl, "api_endpoint", epData, pageMeta{
			Title:       endpoint.Method + " " + endpoint.Path,
			Description: endpoint.Summary,
			Path:        basePath + "/" + endpoint.Slug + "/",
		})
		if err != nil {
			return err
		}

		if err := g.writePage(filepath.Join(outDir, endpoint.Slug, "index.html"), html); err != nil {
			return err
		}
	}

	return nil
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
