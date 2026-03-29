package content

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/mermaid"
)

type TOCEntry struct {
	Level int
	ID    string
	Title string
}

type Page struct {
	Title       string
	Description string
	Date        string
	Author      string
	Draft       bool
	Order       int
	Slug        string
	Section     string
	Group       string
	Content     string
	FilePath    string
	URLPath     string
	Headings    []TOCEntry
	SourcePath  string
}

type PageGroup struct {
	Name  string
	Title string
	Order int
	Nav   []string
	Pages []Page
}

// sortByNav reorders items based on an explicit nav list.
// Items in nav come first (in nav order), the rest follow sorted by Order then Title.
func sortByNav(nav []string, items []Page) []Page {
	if len(nav) == 0 {
		return items
	}

	navIndex := make(map[string]int, len(nav))
	for i, slug := range nav {
		navIndex[slug] = i
	}

	sort.SliceStable(items, func(i, j int) bool {
		ii, iOk := navIndex[items[i].Slug]
		jj, jOk := navIndex[items[j].Slug]

		if iOk && jOk {
			return ii < jj
		}
		if iOk {
			return true
		}
		if jOk {
			return false
		}

		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		return items[i].Title < items[j].Title
	})

	return items
}

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
			&mermaid.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

func ParseMarkdownFile(filePath string) (Page, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Page{}, err
	}
	return ParseMarkdown(data, filePath)
}

func ParseMarkdown(source []byte, filePath string) (Page, error) {
	// Pre-process admonitions (:::type blocks)
	source = processAdmonitions(source)

	var buf bytes.Buffer
	ctx := parser.NewContext()

	if err := md.Convert(source, &buf, parser.WithContext(ctx)); err != nil {
		return Page{}, err
	}

	metadata := meta.Get(ctx)
	page := Page{
		Content:  buf.String(),
		FilePath: filePath,
	}

	if title, ok := metadata["title"].(string); ok {
		page.Title = title
	}
	if desc, ok := metadata["description"].(string); ok {
		page.Description = desc
	}
	if date, ok := metadata["date"].(string); ok {
		page.Date = date
	}
	if author, ok := metadata["author"].(string); ok {
		page.Author = author
	}
	if draft, ok := metadata["draft"].(bool); ok {
		page.Draft = draft
	}
	if order, ok := metadata["order"].(int); ok {
		page.Order = order
	}

	base := filepath.Base(filePath)
	page.Slug = strings.TrimSuffix(base, filepath.Ext(base))
	if page.Title == "" {
		page.Title = strings.ReplaceAll(page.Slug, "-", " ")
		page.Title = strings.Title(page.Title)
	}

	page.Headings = extractHeadings(page.Content)
	page.SourcePath = filePath

	return page, nil
}

func LoadSection(dir string, section string) ([]Page, error) {
	var pages []Page

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return pages, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subPages, err := loadDirPages(filepath.Join(dir, entry.Name()), section, entry.Name())
			if err != nil {
				return nil, err
			}
			pages = append(pages, subPages...)
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "_index.md" {
			continue
		}

		page, err := ParseMarkdownFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}

		if page.Draft {
			continue
		}

		page.Section = section
		page.URLPath = "/" + section + "/" + page.Slug
		pages = append(pages, page)
	}

	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Order != pages[j].Order {
			return pages[i].Order < pages[j].Order
		}
		return pages[i].Title < pages[j].Title
	})

	return pages, nil
}

// parseIndexMeta reads _index.md from a directory and returns frontmatter metadata.
func parseIndexMeta(dir string) (title string, order int, nav []string) {
	indexPath := filepath.Join(dir, "_index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", 0, nil
	}

	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := md.Convert(data, &buf, parser.WithContext(ctx)); err != nil {
		return "", 0, nil
	}

	metadata := meta.Get(ctx)

	if t, ok := metadata["title"].(string); ok {
		title = t
	}
	if o, ok := metadata["order"].(int); ok {
		order = o
	}
	if navRaw, ok := metadata["nav"].([]interface{}); ok {
		for _, item := range navRaw {
			if s, ok := item.(string); ok {
				nav = append(nav, s)
			}
		}
	}

	return title, order, nav
}

func loadDirPages(dir string, section string, group string) ([]Page, error) {
	var pages []Page

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "_index.md" {
			continue
		}

		page, err := ParseMarkdownFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}

		if page.Draft {
			continue
		}

		page.Section = section
		page.Group = group
		page.URLPath = "/" + section + "/" + group + "/" + page.Slug
		pages = append(pages, page)
	}

	return pages, nil
}

func LoadSectionGrouped(dir string, section string) ([]PageGroup, []Page, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	var rootPages []Page
	var groups []PageGroup

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dir, entry.Name())
			subPages, err := loadDirPages(subDir, section, entry.Name())
			if err != nil {
				return nil, nil, err
			}
			if len(subPages) == 0 {
				continue
			}

			groupTitle := strings.ReplaceAll(entry.Name(), "-", " ")
			groupTitle = strings.Title(groupTitle)
			groupOrder := 0
			var groupNav []string

			// Read _index.md for group metadata and nav ordering
			if t, o, n := parseIndexMeta(subDir); t != "" || o != 0 || n != nil {
				if t != "" {
					groupTitle = t
				}
				if o != 0 {
					groupOrder = o
				}
				groupNav = n
			}

			subPages = sortByNav(groupNav, subPages)

			groups = append(groups, PageGroup{
				Name:  entry.Name(),
				Title: groupTitle,
				Order: groupOrder,
				Nav:   groupNav,
				Pages: subPages,
			})
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "_index.md" {
			continue
		}

		page, err := ParseMarkdownFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, nil, err
		}

		if page.Draft {
			continue
		}

		page.Section = section
		page.URLPath = "/" + section + "/" + page.Slug
		rootPages = append(rootPages, page)
	}

	// Read section-level _index.md for nav ordering of pages and groups
	sectionTitle, _, sectionNav := parseIndexMeta(dir)
	_ = sectionTitle // title handled by generator

	rootPages = sortByNav(sectionNav, rootPages)

	if len(sectionNav) > 0 {
		// Sort groups using the same nav list (by group Name)
		navIndex := make(map[string]int, len(sectionNav))
		for i, slug := range sectionNav {
			navIndex[slug] = i
		}
		sort.SliceStable(groups, func(i, j int) bool {
			ii, iOk := navIndex[groups[i].Name]
			jj, jOk := navIndex[groups[j].Name]
			if iOk && jOk {
				return ii < jj
			}
			if iOk {
				return true
			}
			if jOk {
				return false
			}
			if groups[i].Order != groups[j].Order {
				return groups[i].Order < groups[j].Order
			}
			return groups[i].Title < groups[j].Title
		})
	} else {
		sort.Slice(groups, func(i, j int) bool {
			if groups[i].Order != groups[j].Order {
				return groups[i].Order < groups[j].Order
			}
			return groups[i].Title < groups[j].Title
		})
	}

	return groups, rootPages, nil
}

// AllPages returns a flat ordered list: root pages first, then group pages in order.
func AllPages(groups []PageGroup, rootPages []Page) []Page {
	var all []Page
	all = append(all, rootPages...)
	for _, g := range groups {
		all = append(all, g.Pages...)
	}
	return all
}

// admonition types and their display titles
var headingRe = regexp.MustCompile(`<h([2-4])\s+id="([^"]+)"[^>]*>([^<]*(?:<[^/][^>]*>[^<]*</[^>]*>)*[^<]*)</h[2-4]>`)

func extractHeadings(html string) []TOCEntry {
	matches := headingRe.FindAllStringSubmatch(html, -1)
	var entries []TOCEntry
	for _, m := range matches {
		level := 2
		if m[1] == "3" {
			level = 3
		} else if m[1] == "4" {
			level = 4
		}
		// Strip any remaining HTML tags from title
		title := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(m[3], "")
		entries = append(entries, TOCEntry{Level: level, ID: m[2], Title: strings.TrimSpace(title)})
	}
	return entries
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

// processAdmonitions converts :::type / ::: blocks into HTML divs.
func processAdmonitions(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	var result []string
	var stack []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := admonitionOpenRe.FindStringSubmatch(trimmed); matches != nil {
			adType := matches[1]
			customTitle := matches[2]
			title := customTitle
			if title == "" {
				title = admonitionTitles[adType]
			}
			stack = append(stack, adType)
			result = append(result, "")
			result = append(result, `<div class="admonition admonition-`+adType+`">`)
			result = append(result, `<div class="admonition-title">`+title+`</div>`)
			result = append(result, `<div class="admonition-body">`)
			result = append(result, "")
			continue
		}

		if trimmed == ":::" && len(stack) > 0 {
			stack = stack[:len(stack)-1]
			result = append(result, "")
			result = append(result, `</div>`)
			result = append(result, `</div>`)
			result = append(result, "")
			continue
		}

		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n"))
}
