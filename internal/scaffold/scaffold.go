package scaffold

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/omurilo/godoku/internal/config"
)

//go:embed examples/*
var examplesFS embed.FS

func Init(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("Error resolving path: %v", err)
	}

	dirs := []string{
		filepath.Join(absDir, "content", "docs"),
		filepath.Join(absDir, "content", "guides"),
		filepath.Join(absDir, "content", "tutorials"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatalf("Error creating directory %s: %v", d, err)
		}
	}

	if err := os.MkdirAll(filepath.Join(absDir, "apis"), 0755); err != nil {
		log.Fatalf("Error creating apis directory: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.Save(absDir); err != nil {
		log.Fatalf("Error saving config: %v", err)
	}

	copyExample("examples/getting-started.md", filepath.Join(absDir, "content", "docs", "getting-started.md"))
	copyExample("examples/configuration.md", filepath.Join(absDir, "content", "guides", "configuration.md"))
	copyExample("examples/openapi.yaml", filepath.Join(absDir, "apis", "openapi.yaml"))

	fmt.Printf("Godoku project initialized at %s\n", absDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  godoku build    # Build the static site")
	fmt.Println("  godoku serve -w # Start dev server with watch mode")
}

func copyExample(src, dst string) {
	data, err := examplesFS.ReadFile(src)
	if err != nil {
		log.Fatalf("Error reading example %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		log.Fatalf("Error writing %s: %v", dst, err)
	}
}
