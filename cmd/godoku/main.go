package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	godoku "github.com/omurilo/godoku"
	"github.com/omurilo/godoku/internal/config"
	"github.com/omurilo/godoku/internal/generator"
	"github.com/omurilo/godoku/internal/scaffold"
	"github.com/omurilo/godoku/internal/server"
)

const version = "0.2.0"

func main() {
	generator.SetEmbedFS(godoku.AppFS)
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		dir := "."
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		scaffold.Init(dir)
	case "build":
		cmdBuild()
	case "serve":
		cmdServe()
	case "version":
		fmt.Printf("godoku v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`godoku - Static site generator for docs & API references

Usage:
  godoku <command> [options]

Commands:
  init [path]    Initialize a new Godoku project
  build [path]   Build the static site (defaults to the current directory)
  serve [path]   Start a development server (defaults to the current directory)
  version        Show version

Serve Options:
  -p, --port <port>   Port number (default: 3000)
  -w, --watch         Watch for file changes`)
}

func cmdBuild() {
	rootDir, err := resolveRoot(os.Args[2:])
	if err != nil {
		log.Fatalf("Error resolving project directory: %v", err)
	}

	cfg, err := config.Load(rootDir)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	start := time.Now()
	gen := generator.New(cfg, rootDir)
	if err := gen.Build(); err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	fmt.Printf("Site built in %s -> dist/\n", time.Since(start).Round(time.Millisecond))
}

func cmdServe() {
	port := 3000
	watch := false

	var positional []string
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--port":
			if i+1 < len(args) {
				i++
				p, err := strconv.Atoi(args[i])
				if err != nil {
					log.Fatalf("Invalid port: %s", args[i])
				}
				port = p
			}
		case "-w", "--watch":
			watch = true
		default:
			positional = append(positional, args[i])
		}
	}

	rootDir, err := resolveRoot(positional)
	if err != nil {
		log.Fatalf("Error resolving project directory: %v", err)
	}

	cfg, err := config.Load(rootDir)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	srv := server.New(cfg, rootDir, port, watch)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// resolveRoot returns the absolute project directory from the first positional
// argument, defaulting to the current working directory when none is given.
func resolveRoot(args []string) (string, error) {
	dir := "."
	if len(args) > 0 && args[0] != "" {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return abs, nil
}
