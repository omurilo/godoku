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
}

type Branding struct {
	LogoLight string `yaml:"logo_light"`
	LogoDark  string `yaml:"logo_dark"`
	LogoAlt   string `yaml:"logo_alt"`
	LogoLink  string `yaml:"logo_link"`
	Favicon   string `yaml:"favicon"`
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
	return os.WriteFile(filepath.Join(rootDir, "godoku.yaml"), data, 0644)
}
