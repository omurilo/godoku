package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Title       string       `yaml:"title"`
	Description string       `yaml:"description"`
	URL         string       `yaml:"url"`
	Language    string       `yaml:"language"`
	Theme       string       `yaml:"theme"`
	Redirect    string       `yaml:"redirect,omitempty"`
	Branding    Branding     `yaml:"branding"`
	Footer      Footer       `yaml:"footer"`
	Navigation  []NavItem    `yaml:"navigation"`
	Sections    SectionPaths `yaml:"sections"`
	Banner      Banner       `yaml:"banner,omitempty"`
	EditBaseURL string       `yaml:"edit_base_url,omitempty"`
	LLMs        LLMsConfig   `yaml:"llms,omitempty"`
	APIs        []APISpec    `yaml:"apis,omitempty"`
}

// APISpec configures a single OpenAPI specification. Configuring a spec is
// optional: any spec under apis/ is auto-discovered. Listing one here only
// affects ordering (configured specs come first, in this order; the rest follow
// auto-discovered at the end) and lets you override the slug/title/description
// derived from the file name and the spec's info block.
type APISpec struct {
	// Spec is the path to the OpenAPI file. It may be given relative to the
	// project root (e.g. "apis/teams.yaml") or as a bare file name ("teams.yaml")
	// that is matched against the files discovered under apis/.
	Spec        string `yaml:"spec"`
	Slug        string `yaml:"slug,omitempty"`
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Branding struct {
	LogoLight  string `yaml:"logo_light"`
	LogoDark   string `yaml:"logo_dark"`
	LogoAlt    string `yaml:"logo_alt"`
	LogoLink   string `yaml:"logo_link"`
	LogoWidth  string `yaml:"logo_width"`
	LogoHeight string `yaml:"logo_height"`
	Favicon    string `yaml:"favicon"`
}

type NavItem struct {
	Label string `yaml:"label"`
	Path  string `yaml:"path"`
}

type SectionPaths struct {
	Docs      string `yaml:"docs"`
	Guides    string `yaml:"guides"`
	Tutorials string `yaml:"tutorials"`
}

type Footer struct {
	Copyright string         `yaml:"copyright"`
	Position  string         `yaml:"position"`
	Columns   []FooterColumn `yaml:"columns"`
	Social    []FooterSocial `yaml:"social"`
}

type FooterColumn struct {
	Title string       `yaml:"title"`
	Links []FooterLink `yaml:"links"`
}

type FooterLink struct {
	Label string `yaml:"label"`
	Href  string `yaml:"href"`
}

type FooterSocial struct {
	Icon  string `yaml:"icon"`
	Href  string `yaml:"href"`
	Label string `yaml:"label"`
}

type Banner struct {
	Message     string `yaml:"message"`
	Color       string `yaml:"color"`
	Dismissible bool   `yaml:"dismissible"`
}

type LLMsConfig struct {
	LLMsTxt     bool `yaml:"llms_txt"`
	LLMsTxtFull bool `yaml:"llms_txt_full"`
}

func DefaultConfig() Config {
	return Config{
		Title:       "Godoku",
		Description: "API Documentation",
		URL:         "http://localhost:3000",
		Language:    "en",
		Theme:       "default",
		Branding: Branding{
			LogoLink: "/",
		},
		Sections: SectionPaths{
			Docs:      "content/docs",
			Guides:    "content/guides",
			Tutorials: "content/tutorials",
		},
		Navigation: []NavItem{
			{Label: "Docs", Path: "/docs"},
			{Label: "Guides", Path: "/guides"},
			{Label: "Tutorials", Path: "/tutorials"},
			{Label: "API", Path: "/api"},
		},
	}
}

func Load(rootDir string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(filepath.Join(rootDir, "godoku.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Sections.Docs == "" {
		cfg.Sections.Docs = "content/docs"
	}
	if cfg.Sections.Guides == "" {
		cfg.Sections.Guides = "content/guides"
	}
	if cfg.Sections.Tutorials == "" {
		cfg.Sections.Tutorials = "content/tutorials"
	}

	return cfg, nil
}

func (c Config) Save(rootDir string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rootDir, "godoku.yaml"), data, 0o644)
}
