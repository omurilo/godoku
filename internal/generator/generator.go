package generator

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
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
	Config   config.Config
	RootDir  string
	OutDir   string
	NavItems []config.NavItem
}

func New(cfg config.Config, rootDir string) *Generator {
	return &Generator{
		Config:  cfg,
		RootDir: rootDir,
		OutDir:  filepath.Join(rootDir, "public"),
	}
}

func (g *Generator) Build() error {
	if err := os.RemoveAll(g.OutDir); err != nil {
		return fmt.Errorf("cleaning output: %w", err)
	}
	if err := os.MkdirAll(g.OutDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	if err := g.copyStaticAssets(); err != nil {
		return fmt.Errorf("copying static assets: %w", err)
	}

	// Build filtered navigation (hide empty sections)
	g.NavItems = g.buildNavItems()

	if err := g.buildIndex(); err != nil {
		return fmt.Errorf("building index: %w", err)
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

func (g *Generator) loadTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"lower":    strings.ToLower,
		"upper":    strings.ToUpper,
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
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

type layoutData struct {
	Config    config.Config
	NavItems  []config.NavItem
	PageTitle string
	Body      template.HTML
}

func (g *Generator) renderPage(tmpl *template.Template, templateName string, data interface{}, pageTitle string) (string, error) {
	var bodyBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&bodyBuf, templateName, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", templateName, err)
	}

	var pageBuf bytes.Buffer
	ld := layoutData{
		Config:    g.Config,
		NavItems:  g.NavItems,
		PageTitle: pageTitle,
		Body:      template.HTML(bodyBuf.String()),
	}
	if err := tmpl.ExecuteTemplate(&pageBuf, "layout", ld); err != nil {
		return "", fmt.Errorf("executing layout: %w", err)
	}

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

	tmpl, err := g.loadTemplates()
	if err != nil {
		return err
	}

	data := struct {
		Config config.Config
	}{
		Config: g.Config,
	}

	html, err := g.renderPage(tmpl, "index", data, "Home")
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

	html, err := g.renderPage(tmpl, "section", indexData, sectionTitle)
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
		}

		html, err := g.renderPage(tmpl, "section", pageData, page.Title)
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

	html, err := g.renderPage(tmpl, "api_catalog", catalogData, "API Reference")
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

	html, err := g.renderPage(tmpl, "api_index", indexData, doc.Title)
	if err != nil {
		return err
	}

	if err := g.writePage(filepath.Join(outDir, "index.html"), html); err != nil {
		return err
	}

	for _, endpoint := range doc.Endpoints {
		epData := struct {
			Config     config.Config
			Endpoint   openapi.Endpoint
			TagGroups  map[string][]openapi.Endpoint
			ActiveSlug string
			BasePath   string
		}{
			Config:     g.Config,
			Endpoint:   endpoint,
			TagGroups:  doc.TagGroups,
			ActiveSlug: endpoint.Slug,
			BasePath:   basePath,
		}

		html, err := g.renderPage(tmpl, "api_endpoint", epData, endpoint.Method+" "+endpoint.Path)
		if err != nil {
			return err
		}

		if err := g.writePage(filepath.Join(outDir, endpoint.Slug, "index.html"), html); err != nil {
			return err
		}
	}

	return nil
}
