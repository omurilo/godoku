package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/omurilo/godoku/internal/builder"
	"github.com/omurilo/godoku/internal/config"
	"gopkg.in/yaml.v3"
)

// mdxFrontmatter is the metadata block read from the top of a .mdx file. The
// builder strips this same block before compiling the MDX body.
type mdxFrontmatter struct {
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Order       int            `yaml:"order"`
	Draft       bool           `yaml:"draft"`
	Icon        string         `yaml:"icon"`
	SidebarIcon string         `yaml:"sidebar_icon"`
	Nav         []any          `yaml:"nav"`
	API         map[string]any `yaml:"api"`
}

func (fm mdxFrontmatter) navIcon() string {
	if fm.Icon != "" {
		return fm.Icon
	}
	return fm.SidebarIcon
}

// mdxPage is a discovered .md/.mdx file plus its resolved output location.
type mdxPage struct {
	fm         mdxFrontmatter
	Section    string
	Group      string
	Slug       string
	URLPath    string // e.g. "/docs/group/slug"
	OutDir     string // absolute output directory
	SourcePath string
	isIndex    bool // section landing page (_index.md); excluded from sidebar nav
}

func (p mdxPage) title() string {
	if p.fm.Title != "" {
		return p.fm.Title
	}
	t := strings.ReplaceAll(p.Slug, "-", " ")
	return strings.Title(t)
}

type sectionDef struct {
	name string
	dir  string
}

// buildMDXPages renders all documentation through the React/esbuild/sobek MDX
// pipeline: every section page (.md/.mdx), the root content pages (including
// index.md as the homepage), and an auto-generated catalog at each section root
// (/docs/, /guides/, ...). _index.md files are NOT rendered — they only supply
// menu title/ordering. It is a no-op (and touches no network) when there is no
// content, so the React shell is never extracted needlessly.
func (g *Generator) buildMDXPages() error {
	sections := []sectionDef{
		{"docs", g.Config.Sections.Docs},
		{"guides", g.Config.Sections.Guides},
		{"tutorials", g.Config.Sections.Tutorials},
	}

	sectionPages := map[string][]mdxPage{}
	var allSectionPages []mdxPage
	for i, s := range sections {
		dir := s.dir
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(g.RootDir, dir)
		}
		sections[i].dir = dir
		found, err := g.discoverMDX(s.name, dir)
		if err != nil {
			return err
		}
		if len(found) > 0 {
			sectionPages[s.name] = found
			allSectionPages = append(allSectionPages, found...)
		}
	}

	rootPages, err := g.discoverRootContent()
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

	nav := buildMDXNav(g.Config, allSectionPages, sections)
	topNav := buildTopNav(g.Config, sectionPages)
	logo := buildLogo(g.Config)

	// Flat reading order (matching the sidebar) for prev/next page links.
	ordered := flattenPages(sectionPages, g.Config, sections)
	type prevNext struct{ prev, next *mdxPage }
	seq := make(map[string]prevNext, len(ordered))
	for i := range ordered {
		var pn prevNext
		if i > 0 {
			pn.prev = &ordered[i-1]
		}
		if i < len(ordered)-1 {
			pn.next = &ordered[i+1]
		}
		seq[ordered[i].SourcePath] = pn
	}

	baseProps := func() map[string]any {
		p := map[string]any{"nav": nav, "topNav": topNav, "logo": logo}
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

	renderPage := func(p mdxPage) error {
		props := baseProps()
		// Root pages (index.md and other content/*.md) have no section context, so
		// they render full-width without the left navigation sidebar.
		if p.Section == "" {
			props["sidebar"] = false
		}
		if p.fm.API != nil {
			props["api"] = p.fm.API
		} else if toc := tocForFile(p.SourcePath); len(toc) > 0 {
			props["toc"] = toc
		}
		if pn, ok := seq[p.SourcePath]; ok {
			if pn.prev != nil {
				props["prev"] = map[string]any{"title": pn.prev.title(), "href": pageURL(pn.prev.URLPath)}
			}
			if pn.next != nil {
				props["next"] = map[string]any{"title": pn.next.title(), "href": pageURL(pn.next.URLPath)}
			}
		}
		url := pageURL(p.URLPath)
		in := builder.PageInput{
			MDXPath:     p.SourcePath,
			OutDir:      p.OutDir,
			URLPath:     url,
			AssetName:   p.Slug,
			Title:       p.title(),
			Heading:     p.title(),
			Description: p.fm.Description,
			Props:       props,
		}
		if err := b.BuildPage(in); err != nil {
			return fmt.Errorf("rendering %s: %w", p.SourcePath, err)
		}
		g.indexPageMeta(p.title(), p.fm.Description, url, p.Section)
		return nil
	}

	// A single page that fails to compile (e.g. an invalid MDX expression) is
	// logged and skipped rather than aborting the whole build — so one broken
	// page never takes down the dev server or the rest of the site.
	warn := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "godoku: %v\n", err)
		}
	}

	for _, p := range allSectionPages {
		warn(renderPage(p))
	}
	for _, p := range rootPages {
		warn(renderPage(p))
	}

	for _, s := range sections {
		secp := sectionPages[s.name]
		if len(secp) == 0 {
			continue
		}
		warn(g.buildSectionCatalog(b, baseProps(), s, secp))
		// Each category (sub-directory) gets its own catalog at /<section>/<group>/.
		byGroup := map[string][]mdxPage{}
		var groupOrder []string
		for _, p := range secp {
			if p.Group == "" {
				continue
			}
			if _, ok := byGroup[p.Group]; !ok {
				groupOrder = append(groupOrder, p.Group)
			}
			byGroup[p.Group] = append(byGroup[p.Group], p)
		}
		for _, gname := range groupOrder {
			warn(g.buildGroupCatalog(b, baseProps(), s, gname, byGroup[gname]))
		}
	}

	// Homepage: a `redirect` in config, an explicit content/index.md (already
	// rendered above as a root page at "/"), or a synthesized default landing.
	hasIndex := false
	for _, p := range rootPages {
		if pageURL(p.URLPath) == "/" {
			hasIndex = true
			break
		}
	}
	if g.Config.Redirect != "" {
		warn(g.writePage(filepath.Join(g.OutDir, "index.html"), redirectPage(g.Config.Redirect)))
	} else if !hasIndex {
		warn(g.buildDefaultHome(b, baseProps(), sections, sectionPages))
	}

	warn(g.buildMDX404(b, baseProps()))
	g.mdxActive = true

	return nil
}

// buildDefaultHome renders a fallback homepage when the project has no
// content/index.md: a landing with the site title/description and a catalog of
// the sections that have content.
func (g *Generator) buildDefaultHome(b *builder.Builder, props map[string]any, sections []sectionDef, sectionPages map[string][]mdxPage) error {
	props["sidebar"] = false

	items := make([]map[string]any, 0, len(sections))
	for _, s := range sections {
		if len(sectionPages[s.name]) == 0 {
			continue
		}
		title, desc := sectionMeta(s)
		items = append(items, map[string]any{
			"title":       title,
			"description": desc,
			"href":        pageURL("/" + s.name),
		})
	}

	source := ""
	if len(items) > 0 {
		props["catalog"] = items
		props["catalogTitle"] = g.Config.Title
	} else {
		source = "# " + g.Config.Title + "\n\n" + g.Config.Description + "\n"
	}
	props["title"] = g.Config.Title
	props["description"] = g.Config.Description

	if err := b.BuildPage(builder.PageInput{
		Source:      []byte(source),
		OutDir:      g.OutDir,
		URLPath:     "/",
		AssetName:   "index",
		Title:       g.Config.Title,
		Description: g.Config.Description,
		Props:       props,
	}); err != nil {
		return err
	}
	g.indexPageMeta(g.Config.Title, g.Config.Description, "/", "")
	return nil
}

// buildMDX404 renders the 404 page through the React shell (written to
// public/404.html). Assets live at the site root so they resolve from any path.
func (g *Generator) buildMDX404(b *builder.Builder, props map[string]any) error {
	props["sidebar"] = false
	source := "# 404 — Page not found\n\nThe page you are looking for doesn't exist or has been moved.\n\n[Go back home](/)\n"
	in := builder.PageInput{
		Source:      []byte(source),
		OutDir:      g.OutDir,
		OutFile:     "404.html",
		URLPath:     "/",
		AssetName:   "404",
		Title:       "Page Not Found",
		Description: "The page you're looking for doesn't exist.",
		Props:       props,
	}
	return b.BuildPage(in)
}

// indexPageMeta records a rendered page in the sitemap, search index and
// llms.txt collection.
func (g *Generator) indexPageMeta(title, description, url, section string) {
	g.sitemapURLs = append(g.sitemapURLs, url)
	g.searchIndex = append(g.searchIndex, searchEntry{
		Title:       title,
		Description: description,
		Section:     strings.Title(section),
		URL:         url,
	})
	g.llmsEntries = append(g.llmsEntries, llmsEntry{
		Title:       title,
		Description: description,
		URL:         url,
	})
}

// buildSectionCatalog renders /<section>/ as a listing of the section's pages
// (title + description cards). The section's title/description come from its
// _index.md (menu config); there is no Markdown body.
func (g *Generator) buildSectionCatalog(b *builder.Builder, props map[string]any, s sectionDef, pages []mdxPage) error {
	sorted := orderSectionPagesByNav(s.dir, pages)

	items := make([]map[string]any, 0, len(sorted))
	for _, p := range sorted {
		item := map[string]any{
			"title":       p.title(),
			"description": p.fm.Description,
			"href":        pageURL(p.URLPath),
		}
		if p.Group != "" {
			item["group"] = groupLabel(p.Group)
		}
		items = append(items, item)
	}

	title, description := sectionMeta(s)
	props["catalog"] = items
	props["catalogTitle"] = title
	props["title"] = title
	props["description"] = description

	url := "/" + s.name + "/"
	in := builder.PageInput{
		Source:      []byte(""),
		OutDir:      filepath.Join(g.OutDir, s.name),
		URLPath:     url,
		AssetName:   s.name,
		Title:       title,
		Description: description,
		Props:       props,
	}
	if err := b.BuildPage(in); err != nil {
		return fmt.Errorf("rendering %s catalog: %w", s.name, err)
	}
	g.indexPageMeta(title, description, url, s.name)
	return nil
}

// buildGroupCatalog renders /<section>/<group>/ as a listing of the pages in a
// category (sub-directory). Title/description come from the group's _index.md.
func (g *Generator) buildGroupCatalog(b *builder.Builder, props map[string]any, s sectionDef, group string, pages []mdxPage) error {
	sorted := append([]mdxPage(nil), pages...)
	groupNav, _ := indexNavMeta(filepath.Join(s.dir, group))
	sortPagesWithNav(sorted, groupNav)

	items := make([]map[string]any, 0, len(sorted))
	for _, p := range sorted {
		items = append(items, map[string]any{
			"title":       p.title(),
			"description": p.fm.Description,
			"href":        pageURL(p.URLPath),
		})
	}

	title := groupLabel(group)
	description := ""
	if fm, err := parseMDXFrontmatter(filepath.Join(s.dir, group, "_index.md")); err == nil {
		if fm.Title != "" {
			title = fm.Title
		}
		description = fm.Description
	}

	props["catalog"] = items
	props["catalogTitle"] = title
	props["title"] = title
	props["description"] = description

	url := "/" + s.name + "/" + group + "/"
	in := builder.PageInput{
		Source:      []byte(""),
		OutDir:      filepath.Join(g.OutDir, s.name, group),
		URLPath:     url,
		AssetName:   group,
		Title:       title,
		Description: description,
		Props:       props,
	}
	if err := b.BuildPage(in); err != nil {
		return fmt.Errorf("rendering %s/%s catalog: %w", s.name, group, err)
	}
	g.indexPageMeta(title, description, url, s.name)
	return nil
}

// sectionMeta returns a section's display title and description from its
// _index.md frontmatter, falling back to the capitalized section name.
func sectionMeta(s sectionDef) (title, description string) {
	fm, err := parseMDXFrontmatter(filepath.Join(s.dir, "_index.md"))
	if err == nil {
		title = fm.Title
		description = fm.Description
	}
	if title == "" {
		title = strings.Title(s.name)
	}
	return title, description
}

// pageURL normalizes a page's URL path to an absolute, trailing-slash form.
func pageURL(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

// discoverMDX returns the .md/.mdx pages in a section directory, descending one
// level into subdirectories (treated as groups). _index.md is excluded (it is
// menu configuration, handled separately).
func (g *Generator) discoverMDX(section, dir string) ([]mdxPage, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	outBase := filepath.Join(g.OutDir, section)
	var pages []mdxPage

	for _, entry := range entries {
		if entry.IsDir() {
			groupDir := filepath.Join(dir, entry.Name())
			groupEntries, err := os.ReadDir(groupDir)
			if err != nil {
				return nil, err
			}
			for _, ge := range groupEntries {
				if !isContentDoc(ge) {
					continue
				}
				page, ok, err := g.loadMDXPage(filepath.Join(groupDir, ge.Name()), section, entry.Name())
				if err != nil {
					return nil, err
				}
				if ok {
					page.OutDir = filepath.Join(outBase, entry.Name(), page.Slug)
					page.URLPath = "/" + section + "/" + entry.Name() + "/" + page.Slug
					pages = append(pages, page)
				}
			}
			continue
		}

		if !isContentDoc(entry) {
			continue
		}
		page, ok, err := g.loadMDXPage(filepath.Join(dir, entry.Name()), section, "")
		if err != nil {
			return nil, err
		}
		if ok {
			page.OutDir = filepath.Join(outBase, page.Slug)
			page.URLPath = "/" + section + "/" + page.Slug
			pages = append(pages, page)
		}
	}

	return pages, nil
}

// discoverRootContent returns the .md/.mdx pages directly under content/.
// index.md becomes the homepage at /; other files render at /<slug>/.
func (g *Generator) discoverRootContent() ([]mdxPage, error) {
	dir := filepath.Join(g.RootDir, "content")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil // no content dir is fine
	}

	var pages []mdxPage
	for _, entry := range entries {
		if !isContentDoc(entry) {
			continue
		}
		page, ok, err := g.loadMDXPage(filepath.Join(dir, entry.Name()), "", "")
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if page.Slug == "index" {
			page.URLPath = "/"
			page.OutDir = g.OutDir
		} else {
			page.URLPath = "/" + page.Slug
			page.OutDir = filepath.Join(g.OutDir, page.Slug)
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// buildLogo builds the brand logo props from config.Branding, falling back to
// the site title when no logo image is configured.
func buildLogo(cfg config.Config) map[string]any {
	logo := map[string]any{"title": cfg.Title}
	href := "/"
	if cfg.Branding.LogoLink != "" {
		href = cfg.Branding.LogoLink
	}
	logo["href"] = href
	if cfg.Branding.LogoLight != "" {
		logo["srcLight"] = cfg.Branding.LogoLight
	}
	if cfg.Branding.LogoDark != "" {
		logo["srcDark"] = cfg.Branding.LogoDark
	}
	if cfg.Branding.LogoAlt != "" {
		logo["alt"] = cfg.Branding.LogoAlt
	}
	return logo
}

// buildTopNav builds the header navigation from config.Navigation, keeping only
// sections that actually have pages (plus non-section links like /api).
// buildTopNav builds the header navigation straight from the declarative
// godoku.yaml `navigation` list. Section entries (docs/guides/tutorials) with no
// content are hidden so a configured-but-empty section doesn't 404; everything
// else (including a /api entry) is shown exactly as declared.
func buildTopNav(cfg config.Config, sectionPages map[string][]mdxPage) []map[string]any {
	sectionForPath := map[string]string{
		"/docs":      "docs",
		"/guides":    "guides",
		"/tutorials": "tutorials",
	}
	var out []map[string]any
	for _, item := range cfg.Navigation {
		if section, ok := sectionForPath[item.Path]; ok {
			if len(sectionPages[section]) == 0 {
				continue
			}
		}
		out = append(out, map[string]any{"label": item.Label, "href": pageURL(item.Path)})
	}
	return out
}

func (g *Generator) loadMDXPage(path, section, group string) (mdxPage, bool, error) {
	fm, err := parseMDXFrontmatter(path)
	if err != nil {
		return mdxPage{}, false, fmt.Errorf("reading frontmatter of %s: %w", path, err)
	}
	if fm.Draft {
		return mdxPage{}, false, nil
	}
	base := filepath.Base(path)
	slug := strings.TrimSuffix(base, filepath.Ext(base))
	return mdxPage{
		fm:         fm,
		Section:    section,
		Group:      group,
		Slug:       slug,
		SourcePath: path,
	}, true, nil
}

// isContentDoc reports whether a directory entry is a renderable content file:
// a .md or .mdx file other than the special _index.md section page.
func isContentDoc(e os.DirEntry) bool {
	if e.IsDir() {
		return false
	}
	name := e.Name()
	if name == "_index.md" || name == "_index.mdx" {
		return false
	}
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".mdx")
}

// tocForFile builds the table-of-contents entries (h2–h4) for a content file,
// matching the slug ids the React heading components generate at render time.
func tocForFile(path string) []map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return extractTOC(bodyAfterFrontmatter(data))
}

// bodyAfterFrontmatter returns the document body with any leading YAML
// frontmatter block removed.
func bodyAfterFrontmatter(data []byte) []byte {
	text := strings.TrimPrefix(string(data), "\ufeff")
	if !strings.HasPrefix(text, "---") {
		return data
	}
	nl := strings.IndexByte(text, '\n')
	if nl < 0 || strings.TrimSpace(text[:nl]) != "---" {
		return data
	}
	rest := text[nl+1:]
	if i := strings.Index(rest, "\n---"); i >= 0 {
		after := rest[i+len("\n---"):]
		if j := strings.IndexByte(after, '\n'); j >= 0 {
			return []byte(after[j+1:])
		}
		return []byte("")
	}
	return data
}

var tocHeadingRe = regexp.MustCompile(`^(#{2,4})\s+(.+?)\s*#*$`)
var inlineMarkupRe = regexp.MustCompile("[*_`]+")
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// extractTOC scans Markdown for h2–h4 ATX headings (skipping fenced code) and
// returns {level, id, title} maps. id uses slugify, mirroring the heading
// components in the React shell so anchor links line up.
func extractTOC(body []byte) []map[string]any {
	var toc []map[string]any
	inFence := false
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		m := tocHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		title := cleanInlineMarkup(m[2])
		if title == "" {
			continue
		}
		toc = append(toc, map[string]any{
			"level": len(m[1]),
			"id":    slugify(title),
			"title": title,
		})
	}
	return toc
}

func cleanInlineMarkup(s string) string {
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = inlineMarkupRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// slugify lowercases and hyphenates a heading. It MUST stay byte-for-byte
// compatible with the slugify() in app/lib/slug.ts so TOC hrefs match the ids
// rendered on the headings.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if b.Len() > 0 && !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// parseMDXFrontmatter extracts and decodes the leading YAML frontmatter block.
// A file with no frontmatter yields a zero-value struct and no error.
func parseMDXFrontmatter(path string) (mdxFrontmatter, error) {
	var fm mdxFrontmatter
	data, err := os.ReadFile(path)
	if err != nil {
		return fm, err
	}
	block, ok := frontmatterBlock(data)
	if !ok {
		return fm, nil
	}
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		return fm, fmt.Errorf("parsing yaml frontmatter: %w", err)
	}
	return fm, nil
}

// frontmatterBlock returns the YAML text between the leading "---" fences.
func frontmatterBlock(data []byte) (string, bool) {
	text := strings.TrimPrefix(string(data), "\ufeff")
	if !strings.HasPrefix(text, "---") {
		return "", false
	}
	nl := strings.IndexByte(text, '\n')
	if nl < 0 || strings.TrimSpace(text[:nl]) != "---" {
		return "", false
	}
	rest := text[nl+1:]
	lines := strings.Split(rest, "\n")
	var block []string
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			_ = i
			return strings.Join(block, "\n"), true
		}
		block = append(block, line)
	}
	return "", false
}

// buildMDXNav builds the sidebar tree: one collapsible group per navigated
// section, in the order declared in godoku.yaml's navigation, with that
// section's pages (and any subgroups) as nested links. The result is a plain
// []map serialized to JSON and consumed by the React shell's `NavLink[]` type.
func buildMDXNav(cfg config.Config, pages []mdxPage, sections []sectionDef) []map[string]any {
	bySection := map[string][]mdxPage{}
	for _, p := range pages {
		bySection[p.Section] = append(bySection[p.Section], p)
	}

	dirOf := map[string]string{}
	for _, s := range sections {
		dirOf[s.name] = s.dir
	}

	sectionForPath := map[string]string{
		"/docs":      "docs",
		"/guides":    "guides",
		"/tutorials": "tutorials",
	}

	var nav []map[string]any
	seen := map[string]bool{}
	add := func(label, section string) {
		secPages := bySection[section]
		if len(secPages) == 0 || seen[section] {
			return
		}
		seen[section] = true
		nav = append(nav, sectionNav(label, dirOf[section], secPages))
	}

	for _, item := range cfg.Navigation {
		if section, ok := sectionForPath[item.Path]; ok {
			add(item.Label, section)
		}
	}
	for _, section := range []string{"docs", "guides", "tutorials"} {
		add(strings.Title(section), section)
	}
	return nav
}

// indexIcon returns the `icon` frontmatter of a directory's _index.md, or "".
func indexIcon(dir string) string {
	if dir == "" {
		return ""
	}
	fm, err := parseMDXFrontmatter(filepath.Join(dir, "_index.md"))
	if err != nil {
		return ""
	}
	return fm.navIcon()
}

// indexTitle returns the `title` frontmatter of a directory's _index.md, or
// fallback when missing.
func indexTitle(dir, fallback string) string {
	if dir == "" {
		return fallback
	}
	fm, err := parseMDXFrontmatter(filepath.Join(dir, "_index.md"))
	if err != nil || fm.Title == "" {
		return fallback
	}
	return fm.Title
}

func sectionNav(label, dir string, pages []mdxPage) map[string]any {
	var root []mdxPage
	groups := map[string][]mdxPage{}
	var groupOrder []string
	for _, p := range pages {
		if p.isIndex {
			continue // section landing page is not a sidebar leaf
		}
		if p.Group == "" {
			root = append(root, p)
			continue
		}
		if _, ok := groups[p.Group]; !ok {
			groupOrder = append(groupOrder, p.Group)
		}
		groups[p.Group] = append(groups[p.Group], p)
	}

	navOrder, navIcons := indexNavMeta(dir)
	sortPagesWithNav(root, navOrder)
	items := make([]map[string]any, 0, len(root)+len(groupOrder))

	rootBySlug := make(map[string]mdxPage, len(root))
	for _, p := range root {
		rootBySlug[p.Slug] = p
	}
	usedRoot := map[string]bool{}
	usedGroup := map[string]bool{}

	appendGroup := func(gname string) {
		gp := groups[gname]
		if len(gp) == 0 {
			return
		}
		groupDir := filepath.Join(dir, gname)
		groupNavOrder, groupNavIcons := indexNavMeta(groupDir)
		sortPagesWithNav(gp, groupNavOrder)
		children := make([]map[string]any, 0, len(gp))
		for _, p := range gp {
			children = append(children, leaf(p, groupNavIcons[p.Slug]))
		}
		entry := map[string]any{
			"label": indexTitle(groupDir, groupLabel(gname)),
			"items": children,
		}
		if len(gp) > 0 {
			entry["href"] = pageURL("/" + gp[0].Section + "/" + gname)
		}
		if dir != "" {
			if ic := indexIcon(groupDir); ic != "" {
				entry["icon"] = ic
			}
		}
		items = append(items, entry)
	}

	// Respect explicit nav order for both page slugs and group slugs.
	for _, slug := range navOrder {
		if p, ok := rootBySlug[slug]; ok {
			items = append(items, leaf(p, navIcons[p.Slug]))
			usedRoot[slug] = true
			continue
		}
		if _, ok := groups[slug]; ok {
			appendGroup(slug)
			usedGroup[slug] = true
		}
	}

	// Append any remaining root pages in default order.
	for _, p := range root {
		if usedRoot[p.Slug] {
			continue
		}
		items = append(items, leaf(p, navIcons[p.Slug]))
	}

	sort.Strings(groupOrder)
	for _, gname := range groupOrder {
		if usedGroup[gname] {
			continue
		}
		appendGroup(gname)
	}

	out := map[string]any{"label": label, "items": items}
	if ic := indexIcon(dir); ic != "" {
		out["icon"] = ic
	}
	return out
}

// flattenPages returns all section pages in sidebar reading order (sections in
// navigation order; within a section: root pages, then each group's pages),
// used to compute prev/next links.
func flattenPages(sectionPages map[string][]mdxPage, cfg config.Config, sections []sectionDef) []mdxPage {
	sectionForPath := map[string]string{"/docs": "docs", "/guides": "guides", "/tutorials": "tutorials"}
	sectionDir := map[string]string{}
	for _, s := range sections {
		sectionDir[s.name] = s.dir
	}
	var order []string
	seen := map[string]bool{}
	for _, item := range cfg.Navigation {
		if s, ok := sectionForPath[item.Path]; ok && !seen[s] && len(sectionPages[s]) > 0 {
			order = append(order, s)
			seen[s] = true
		}
	}
	for _, s := range []string{"docs", "guides", "tutorials"} {
		if !seen[s] && len(sectionPages[s]) > 0 {
			order = append(order, s)
			seen[s] = true
		}
	}

	var flat []mdxPage
	for _, s := range order {
		flat = append(flat, orderSectionPagesByNav(sectionDir[s], sectionPages[s])...)
	}
	return flat
}

func groupLabel(name string) string {
	if name == "" {
		return ""
	}
	return strings.Title(strings.ReplaceAll(name, "-", " "))
}

func leaf(p mdxPage, overrideIcon string) map[string]any {
	out := map[string]any{"label": p.title(), "href": pageURL(p.URLPath)}
	icon := p.fm.navIcon()
	if overrideIcon != "" {
		icon = overrideIcon
	}
	if icon != "" {
		out["icon"] = icon
	}
	return out
}

// indexNavMeta returns nav ordering and icon overrides from a directory's
// _index.md frontmatter. Supports mixed formats:
// - nav: ["getting-started", "configuration"]
// - nav: [{ slug: "getting-started", icon: "rocket" }]
func indexNavMeta(dir string) ([]string, map[string]string) {
	icons := map[string]string{}
	if dir == "" {
		return nil, icons
	}
	fm, err := parseMDXFrontmatter(filepath.Join(dir, "_index.md"))
	if err != nil {
		return nil, icons
	}
	if len(fm.Nav) == 0 {
		return nil, icons
	}
	order := make([]string, 0, len(fm.Nav))
	for _, item := range fm.Nav {
		switch v := item.(type) {
		case string:
			if v != "" {
				order = append(order, v)
			}
		case map[string]any:
			slug, _ := v["slug"].(string)
			if slug == "" {
				continue
			}
			order = append(order, slug)
			if icon, ok := v["icon"].(string); ok && icon != "" {
				icons[slug] = icon
			}
		}
	}
	return order, icons
}

func sortPagesWithNav(ps []mdxPage, nav []string) {
	sortPages(ps)
	if len(nav) == 0 {
		return
	}
	pos := make(map[string]int, len(nav))
	for i, slug := range nav {
		pos[slug] = i
	}
	sort.SliceStable(ps, func(i, j int) bool {
		ii, iok := pos[ps[i].Slug]
		jj, jok := pos[ps[j].Slug]
		if iok && jok {
			return ii < jj
		}
		if iok {
			return true
		}
		if jok {
			return false
		}
		if ps[i].fm.Order != ps[j].fm.Order {
			return ps[i].fm.Order < ps[j].fm.Order
		}
		return ps[i].title() < ps[j].title()
	})
}

func sortPages(ps []mdxPage) {
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].fm.Order != ps[j].fm.Order {
			return ps[i].fm.Order < ps[j].fm.Order
		}
		return ps[i].title() < ps[j].title()
	})
}

// orderSectionPagesByNav returns a section's pages in the same order the sidebar
// uses: the _index.md `nav` list drives both root pages and groups (groups
// expand to their pages, ordered by the group's own _index.md nav); anything
// not listed follows in default order. Used by the section catalog and
// prev/next so all three agree.
func orderSectionPagesByNav(dir string, pages []mdxPage) []mdxPage {
	var root []mdxPage
	groups := map[string][]mdxPage{}
	var groupOrder []string
	for _, p := range pages {
		if p.isIndex {
			continue
		}
		if p.Group == "" {
			root = append(root, p)
			continue
		}
		if _, ok := groups[p.Group]; !ok {
			groupOrder = append(groupOrder, p.Group)
		}
		groups[p.Group] = append(groups[p.Group], p)
	}

	navOrder, _ := indexNavMeta(dir)
	sortPagesWithNav(root, navOrder)
	rootBySlug := make(map[string]mdxPage, len(root))
	for _, p := range root {
		rootBySlug[p.Slug] = p
	}
	usedRoot := map[string]bool{}
	usedGroup := map[string]bool{}

	var out []mdxPage
	appendGroup := func(gname string) {
		gp := groups[gname]
		if len(gp) == 0 {
			return
		}
		gno, _ := indexNavMeta(filepath.Join(dir, gname))
		sortPagesWithNav(gp, gno)
		out = append(out, gp...)
	}

	for _, slug := range navOrder {
		if p, ok := rootBySlug[slug]; ok {
			out = append(out, p)
			usedRoot[slug] = true
			continue
		}
		if _, ok := groups[slug]; ok {
			appendGroup(slug)
			usedGroup[slug] = true
		}
	}
	for _, p := range root {
		if !usedRoot[p.Slug] {
			out = append(out, p)
		}
	}
	sort.Strings(groupOrder)
	for _, gname := range groupOrder {
		if !usedGroup[gname] {
			appendGroup(gname)
		}
	}
	return out
}

// materializeApp extracts the embedded React shell to a temp directory so
// esbuild can resolve its files on disk. The path is cached on the Generator
// and reused across pages (and rebuilds during `serve --watch`).
func (g *Generator) materializeApp() (string, error) {
	if g.appDir != "" {
		if _, err := os.Stat(g.appDir); err == nil {
			return g.appDir, nil
		}
	}

	dst, err := os.MkdirTemp("", "godoku-app-")
	if err != nil {
		return "", err
	}

	err = fs.WalkDir(appFS, "app", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel("app", p)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := appFS.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return "", fmt.Errorf("walking embedded app fs: %w", err)
	}

	g.appDir = dst
	return dst, nil
}
