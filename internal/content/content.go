package content

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
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

	// Pre-process code block metadata (extract title, line highlights, line numbers)
	source, codeBlockMeta := extractCodeBlockMeta(source)

	var buf bytes.Buffer
	ctx := parser.NewContext()

	if err := md.Convert(source, &buf, parser.WithContext(ctx)); err != nil {
		return Page{}, err
	}

	rendered := buf.String()

	// Post-process: apply code block enhancements
	if len(codeBlockMeta) > 0 {
		rendered = applyCodeBlockMeta(rendered, codeBlockMeta)
	}

	metadata := meta.Get(ctx)
	page := Page{
		Content:  rendered,
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

// Code block metadata
type codeBlockInfo struct {
	index        int
	lang         string
	title        string
	highlightStr string
	lineNumbers  bool
}

var fenceOpenRe = regexp.MustCompile("^```(\\w*)(.*?)\\s*$")
var titleRe = regexp.MustCompile(`title="([^"]*)"`)
var highlightRe = regexp.MustCompile(`\{([0-9,\-\s]+)\}`)

// extractCodeBlockMeta scans markdown source for fenced code blocks with metadata
// like ```js {1,3-5} title="file.js" showLineNumbers and strips the metadata
// so goldmark can render the code normally.
func extractCodeBlockMeta(source []byte) ([]byte, []codeBlockInfo) {
	lines := strings.Split(string(source), "\n")
	var result []string
	var metas []codeBlockInfo
	blockIndex := 0
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			if matches := fenceOpenRe.FindStringSubmatch(trimmed); matches != nil {
				inBlock = true
				lang := matches[1]
				rest := matches[2]

				var info codeBlockInfo
				info.index = blockIndex
				info.lang = lang
				hasMeta := false

				// Extract title
				if titleMatch := titleRe.FindStringSubmatch(rest); titleMatch != nil {
					info.title = titleMatch[1]
					hasMeta = true
				}

				// Extract line highlights {1,3-5}
				if hlMatch := highlightRe.FindStringSubmatch(rest); hlMatch != nil {
					info.highlightStr = hlMatch[1]
					hasMeta = true
				}

				// Extract showLineNumbers
				if strings.Contains(rest, "showLineNumbers") {
					info.lineNumbers = true
					hasMeta = true
				}

				// Always add if lang is present, even if no extra meta
				if hasMeta || lang != "" {
					metas = append(metas, info)
				}

				// Write clean fence line (only language)
				result = append(result, "```"+lang)
				continue
			}
		} else if trimmed == "```" {
			inBlock = false
			blockIndex++
			result = append(result, line)
			continue
		}

		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n")), metas
}

var preBlockRe = regexp.MustCompile(`(?s)<pre[^>]*>.*?</pre>`)

// applyCodeBlockMeta injects data attributes on <pre> tags for client-side Shiki processing,
// and wraps blocks with a title div when present.
func applyCodeBlockMeta(html string, metas []codeBlockInfo) string {
	metaMap := make(map[int]codeBlockInfo)
	for _, m := range metas {
		metaMap[m.index] = m
	}

	blockIdx := 0
	html = preBlockRe.ReplaceAllStringFunc(html, func(block string) string {
		info, hasMeta := metaMap[blockIdx]
		blockIdx++

		if !hasMeta {
			return block
		}

		// Build data attributes for Shiki
		var attrs []string
		if info.lang != "" {
			attrs = append(attrs, `data-lang="`+info.lang+`"`)
		}
		if info.highlightStr != "" {
			attrs = append(attrs, `data-highlight="`+info.highlightStr+`"`)
		}
		if info.lineNumbers {
			attrs = append(attrs, `data-line-numbers="true"`)
		}
		if len(attrs) > 0 {
			block = strings.Replace(block, "<pre", "<pre "+strings.Join(attrs, " "), 1)
		}

		// Wrap with title if present
		if info.title != "" {
			block = `<div class="code-block-titled"><div class="code-block-title">` + escapeHTML(info.title) + `</div>` + block + `</div>`
		}

		return block
	})

	return html
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
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
