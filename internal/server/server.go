package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/omurilo/godoku/internal/config"
	"github.com/omurilo/godoku/internal/generator"
)

type Server struct {
	Config  config.Config
	RootDir string
	Port    int
	Watch   bool
}

func New(cfg config.Config, rootDir string, port int, watch bool) *Server {
	return &Server{
		Config:  cfg,
		RootDir: rootDir,
		Port:    port,
		Watch:   watch,
	}
}

func (s *Server) Start() error {
	gen := generator.New(s.Config, s.RootDir)
	if err := gen.Build(); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	if s.Watch {
		go s.watchFiles(gen)
	}

	publicDir := filepath.Join(s.RootDir, "public")
	fileServer := http.FileServer(http.Dir(publicDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/" && s.Config.Redirect != "" {
			http.Redirect(w, r, s.Config.Redirect, http.StatusMovedPermanently)
			return
		}

		fullPath := filepath.Join(publicDir, filepath.Clean(path))

		if !strings.HasSuffix(path, "/") && filepath.Ext(path) == "" {
			indexPath := filepath.Join(fullPath, "index.html")
			if fileExists(indexPath) {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		// Check if the requested file exists
		if filepath.Ext(path) != "" {
			if fileExists(fullPath) {
				fileServer.ServeHTTP(w, r)
				return
			}
		} else if strings.HasSuffix(path, "/") {
			indexPath := filepath.Join(fullPath, "index.html")
			if fileExists(indexPath) {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Serve custom 404 page
		notFoundPath := filepath.Join(publicDir, "404.html")
		if fileExists(notFoundPath) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			data, _ := os.ReadFile(notFoundPath)
			w.Write(data)
			return
		}

		http.NotFound(w, r)
	})

	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("Godoku dev server running at http://localhost:%d", s.Port)
	if s.Watch {
		log.Println("Watching for file changes...")
	}

	return http.ListenAndServe(addr, mux)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Server) watchFiles(gen *generator.Generator) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error creating watcher: %v", err)
		return
	}
	defer watcher.Close()

	watchDirs := []string{
		filepath.Join(s.RootDir, s.Config.Sections.Docs),
		filepath.Join(s.RootDir, s.Config.Sections.Guides),
		filepath.Join(s.RootDir, s.Config.Sections.Tutorials),
	}

	for _, dir := range watchDirs {
		if err := watcher.Add(dir); err != nil {
			log.Printf("Warning: cannot watch %s: %v", dir, err)
		}
	}

	// Watch apis/ directory
	apisDir := filepath.Join(s.RootDir, "apis")
	if err := watcher.Add(apisDir); err != nil {
		log.Printf("Warning: cannot watch %s: %v", apisDir, err)
	}

	if err := watcher.Add(s.RootDir); err != nil {
		log.Printf("Warning: cannot watch root dir: %v", err)
	}

	var lastBuild time.Time
	debounce := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				if time.Since(lastBuild) < debounce {
					continue
				}
				lastBuild = time.Now()

				log.Printf("Change detected: %s", event.Name)

				if filepath.Base(event.Name) == "godoku.yaml" {
					newCfg, err := config.Load(s.RootDir)
					if err != nil {
						log.Printf("Error reloading config: %v", err)
						continue
					}
					s.Config = newCfg
					gen = generator.New(newCfg, s.RootDir)
				}

				if err := gen.Build(); err != nil {
					log.Printf("Rebuild failed: %v", err)
				} else {
					log.Println("Rebuild complete")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
