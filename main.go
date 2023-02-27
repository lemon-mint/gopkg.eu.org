package main

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lemon-mint/gopkg.eu.org/types"
	"github.com/lemon-mint/gopkg.eu.org/views"
	"github.com/otiai10/copy"
)

const SOURCE_PATH = "./modules"
const DIST_PATH = "./dist"
const STATIC_PATH = "./public"

type VCS string

type Module = types.Module

func parseSource(m *Module) error {
	/* Example File Format:

	// This is a comment
	# This is also a comment
	root: gopkg.eu.org/example
	vcs: git
	url: https://example.com/jane/example.git
	description: Example Module providing a root access to the hello world example.
	*/

	file, err := os.Open(filepath.Join(SOURCE_PATH, m.Path))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "root":
			m.Root = value
		case "vcs":
			m.VCS = value
		case "url":
			m.RepoURL = value
		case "description":
			m.Description = value
		}
	}

	// Handle Default Values
	if m.VCS == "" {
		m.VCS = "git"
	}

	return nil
}

func renderTemplate(m *Module) error {
	dir := filepath.Dir(m.Path)
	var index string
	if dir == "." {
		index = filepath.Join(DIST_PATH, m.Path+".html")
	} else {
		base := filepath.Base(m.Path)
		index = filepath.Join(DIST_PATH, dir, base+".html")
	}
	file, err := os.Create(index)
	if err != nil {
		return err
	}
	defer file.Close()

	// Render Template
	views.WriteMod(file, m.Root, m.VCS, m.RepoURL, m.Description)

	return nil
}

var worker_count int = runtime.NumCPU()
var worker_id uint32
var wg sync.WaitGroup

func parseWorker(queue chan *Module) {
	defer wg.Done()

	id := atomic.AddUint32(&worker_id, 1)
	for m := range queue {
		err := parseSource(m)
		if err != nil {
			log.Printf("Worker %d: %s", id, err)
		}
	}
}

func templateWorker(queue chan *Module) {
	defer wg.Done()

	id := atomic.AddUint32(&worker_id, 1)
	for m := range queue {
		err := renderTemplate(m)
		if err != nil {
			log.Printf("Worker %d: %s", id, err)
		}
	}
}

//go:generate go get -u github.com/valyala/quicktemplate/qtc
//go:generate go run github.com/valyala/quicktemplate/qtc -dir=views
//go:generate go mod tidy

func main() {
	var modules []*Module
	queue := make(chan *Module, worker_count)

	// Start Workers
	wg.Add(worker_count)
	for i := 0; i < worker_count; i++ {
		go parseWorker(queue)
	}

	filepath.WalkDir(
		SOURCE_PATH,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Ignore Directories
			if d.IsDir() {
				return nil
			}

			// Ignore dotfiles
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}

			// Calculate Relative Path
			rel_path, err := filepath.Rel(SOURCE_PATH, path)
			if err != nil {
				return err
			}

			// Add Module to Indexing Queue
			m := &Module{
				Path: rel_path,
			}
			modules = append(modules, m)
			queue <- m

			return nil
		},
	)

	// Stop Workers
	close(queue)
	// Wait for Workers to finish
	wg.Wait()

	// Sort Modules by Root
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Root < modules[j].Root
	})

	// Clear Dist Directory
	err := os.RemoveAll(DIST_PATH)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(DIST_PATH, 0755)
	if err != nil {
		log.Fatal(err)
	}

	queue = make(chan *Module, worker_count)
	wg.Add(worker_count)
	for i := 0; i < worker_count; i++ {
		go templateWorker(queue)
	}

	// Generate Index
	for i := range modules {
		// log.Printf("module: %s", modules[i].Root)
		// log.Printf("  path: %s", modules[i].Path)
		// log.Printf("   vcs: %s", modules[i].VCS)
		// log.Printf("   url: %s\n", modules[i].RepoURL)

		dir := filepath.Dir(modules[i].Path)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(DIST_PATH, dir), 0755)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Render Template
		queue <- modules[i]
	}

	// Stop Workers
	close(queue)
	// Wait for Workers to finish
	wg.Wait()

	// Copy Static Files
	err = copy.Copy(STATIC_PATH, DIST_PATH)
	if err != nil {
		log.Fatal(err)
	}

	// Generate Index
	index := filepath.Join(DIST_PATH, "index.html")
	file, err := os.Create(index)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Render Template
	views.WriteIndex(file, modules)
}
