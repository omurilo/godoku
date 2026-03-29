package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	godoku "github.com/omurilo/godoku"
	"github.com/omurilo/godoku/internal/config"
	"github.com/omurilo/godoku/internal/generator"
	"github.com/omurilo/godoku/internal/scaffold"
	"github.com/omurilo/godoku/internal/server"
)

const version = "0.1.0"

func main() {
	generator.SetEmbedFS(godoku.TemplatesFS, godoku.StaticFS)
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
  build          Build the static site
  serve          Start a development server
  version        Show version

Serve Options:
  -p, --port <port>   Port number (default: 3000)
  -w, --watch         Watch for file changes`)
}

func cmdBuild() {
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %v", err)
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

	fmt.Printf("Site built in %s -> public/\n", time.Since(start).Round(time.Millisecond))
}

func cmdServe() {
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory: %v", err)
	}

	cfg, err := config.Load(rootDir)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	port := 3000
	watch := false

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
		}
	}

	srv := server.New(cfg, rootDir, port, watch)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
